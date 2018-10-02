package codegen

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"goa.design/goa/expr"
)

var (
	transformArrayT *template.Template
	transformMapT   *template.Template
)

type (
	// Transformer interface defines the functions to transform a source
	// attribute expression to a target attribute expression.
	Transformer interface {
		// TransformAttribute transforms the src attribute to tgt.
		TransformAttribute(src, tgt *expr.AttributeExpr, srcVar, tgtVar, srcPkg, tgtPkg string, newVar bool) (string, error)
		// TransformAttributeHelpers returns the helper functions required to
		// transform src data type to tgt.
		TransformAttributeHelpers(src, tgt expr.DataType, srcPkg, tgtPkg string, seen ...map[string]*TransformFunctionData) ([]*TransformFunctionData, error)
		// TransformFunctionData returns the TransformFunctionData for the given
		// source and target attributes and code.
		TransformFunctionData(src, tgt *expr.AttributeExpr, srcPkg, tgtPkg, code string) *TransformFunctionData
		// Helper returns the name of the helper function used to transform
		// src to tgt.
		Helper(src, tgt *expr.AttributeExpr) string
	}

	// TransformFunctionData describes a helper function used to transform
	// user types. These are necessary to prevent potential infinite
	// recursion when a type attribute is defined recursively. For example:
	//
	//     var Recursive = Type("Recursive", func() {
	//         Attribute("r", "Recursive")
	//     }
	//
	// Transforming this type requires generating an intermediary function:
	//
	//     func recursiveToRecursive(r *Recursive) *service.Recursive {
	//         var t service.Recursive
	//         if r.R != nil {
	//             t.R = recursiveToRecursive(r.R)
	//         }
	//    }
	//
	TransformFunctionData struct {
		Name          string
		ParamTypeRef  string
		ResultTypeRef string
		Code          string
	}

	goTransformer struct {
		// unmarshal if true indicates whether the code is being generated to
		// initialize a type from unmarshaled data, otherwise to initialize a type that
		// is marshaled:
		//
		//   if unmarshal is true (unmarshal)
		//     - The source type uses pointers for all fields - even required ones.
		//     - The target type do not use pointers for primitive fields that have
		//       default values even when not required.
		//
		//   if unmarshal is false (marshal)
		//     - The source type used do not use pointers for primitive fields that
		//       have default values even when not required.
		//     - The target type fields are initialized with their default values
		//       (if any) when source field is a primitive pointer and nil or a
		//       non-primitive type and nil
		unmarshal bool
		scope     *NameScope
	}
)

// NOTE: can't initialize inline because https://github.com/golang/go/issues/1817
func init() {
	funcMap := template.FuncMap{"transformAttribute": transformAttributeHelper}
	transformArrayT = template.Must(template.New("transformArray").Funcs(funcMap).Parse(transformArrayTmpl))
	transformMapT = template.Must(template.New("transformMap").Funcs(funcMap).Parse(transformMapTmpl))
}

// GoTypeTransform produces Go code that initializes the data structure defined
// by target from an instance of the data structure described by source. The
// data structures can be objects, arrays or maps. The algorithm matches object
// fields by name and ignores object fields in target that don't have a match in
// source. The matching and generated code leverage mapped attributes so that
// attribute names may use the "name:elem" syntax to define the name of the
// design attribute and the name of the corresponding generated Go struct field.
// The function returns an error if target is not compatible with source
// (different type, fields of different type etc).
//
// sourceVar and targetVar contain the name of the variables that hold the
// source and target data structures respectively.
//
// sourcePkg and targetPkg contain the name of the Go package that defines the
// source or target type respectively in case it's not the same package as where
// the generated code lives.
//
// unmarshal if true indicates whether the code is being generated to
// initialize a type from unmarshaled data, otherwise to initialize a type that
// is marshaled:
//
//   if unmarshal is true (unmarshal)
//     - The source type uses pointers for all fields - even required ones.
//     - The target type do not use pointers for primitive fields that have
//			 default values even when not required.
//
//   if unmarshal is false (marshal)
//     - The source type used do not use pointers for primitive fields that
//			 have default values even when not required.
//     - The target type fields are initialized with their default values
//			 (if any) when source field is a primitive pointer and nil or a
//			 non-primitive type and nil.
//
// scope is used to compute the name of the user types when initializing fields
// that use them.
//
func GoTypeTransform(source, target expr.DataType, sourceVar, targetVar, sourcePkg, targetPkg string, unmarshal bool, scope *NameScope) (string, []*TransformFunctionData, error) {

	var (
		satt = &expr.AttributeExpr{Type: source}
		tatt = &expr.AttributeExpr{Type: target}
	)

	g := &goTransformer{unmarshal: unmarshal, scope: scope}

	code, err := g.TransformAttribute(satt, tatt, sourceVar, targetVar, sourcePkg, targetPkg, true)
	if err != nil {
		return "", nil, err
	}

	funcs, err := g.TransformAttributeHelpers(source, target, sourcePkg, targetPkg)
	if err != nil {
		return "", nil, err
	}

	return strings.TrimRight(code, "\n"), funcs, nil
}

// TransformAttribute transforms source attribute to target.
func (g *goTransformer) TransformAttribute(source, target *expr.AttributeExpr, sourceVar, targetVar, sourcePkg, targetPkg string, newVar bool) (string, error) {
	if err := IsCompatible(source.Type, target.Type, sourceVar, targetVar); err != nil {
		return "", err
	}
	var (
		code string
		err  error
	)
	switch {
	case expr.IsArray(source.Type):
		code, err = g.transformArray(expr.AsArray(source.Type), expr.AsArray(target.Type), sourceVar, targetVar, sourcePkg, targetPkg, newVar)
	case expr.IsMap(source.Type):
		code, err = g.transformMap(expr.AsMap(source.Type), expr.AsMap(target.Type), sourceVar, targetVar, sourcePkg, targetPkg, newVar)
	case expr.IsObject(source.Type):
		code, err = g.transformObject(source, target, sourceVar, targetVar, sourcePkg, targetPkg, newVar)
	default:
		assign := "="
		if newVar {
			assign = ":="
		}
		if _, ok := target.Type.(expr.UserType); ok {
			// Primitive user type, these are used for error results
			cast := g.scope.GoFullTypeRef(target, targetPkg)
			return fmt.Sprintf("%s %s %s(%s)\n", targetVar, assign, cast, sourceVar), nil
		}
		code = fmt.Sprintf("%s %s %s\n", targetVar, assign, sourceVar)
	}
	if err != nil {
		return "", err
	}
	return code, nil
}

// TransformAttributeHelpers returns the transform functions required to
// transform source data type to target. It returns an error if source and
// target are not compatible.
func (g *goTransformer) TransformAttributeHelpers(source, target expr.DataType, sourcePkg, targetPkg string, seen ...map[string]*TransformFunctionData) ([]*TransformFunctionData, error) {
	var (
		helpers []*TransformFunctionData
		err     error
	)
	// Do not generate a transform function for the top most user type.
	switch {
	case expr.IsArray(source):
		source = expr.AsArray(source).ElemType.Type
		target = expr.AsArray(target).ElemType.Type
		helpers, err = g.TransformAttributeHelpers(source, target, sourcePkg, targetPkg, seen...)
	case expr.IsMap(source):
		sm := expr.AsMap(source)
		tm := expr.AsMap(target)
		source = sm.ElemType.Type
		target = tm.ElemType.Type
		helpers, err = g.TransformAttributeHelpers(source, target, sourcePkg, targetPkg, seen...)
		if err == nil {
			var other []*TransformFunctionData
			source = sm.KeyType.Type
			target = tm.KeyType.Type
			other, err = g.TransformAttributeHelpers(source, target, sourcePkg, targetPkg, seen...)
			helpers = append(helpers, other...)
		}
	case expr.IsObject(source):
		helpers, err = TransformObjectHelpers(source, target, sourcePkg, targetPkg, g, seen...)
	}
	if err != nil {
		return nil, err
	}
	return helpers, nil
}

func (g *goTransformer) Helper(src, tgt *expr.AttributeExpr) string {
	var (
		sname  string
		tname  string
		prefix string
	)
	{
		sname = g.scope.GoTypeName(src)
		if _, ok := src.Meta["goa.external"]; ok {
			// type belongs to external package so name could clash
			sname += "Ext"
		}
		tname = g.scope.GoTypeName(tgt)
		if _, ok := tgt.Meta["goa.external"]; ok {
			// type belongs to external package so name could clash
			tname += "Ext"
		}
		prefix = "marshal"
		if g.unmarshal {
			prefix = "unmarshal"
		}
	}
	return Goify(prefix+sname+"To"+tname, false)
}

func (g *goTransformer) TransformFunctionData(source, target *expr.AttributeExpr, sourcePkg, targetPkg, code string) *TransformFunctionData {
	return &TransformFunctionData{
		Name:          g.Helper(source, target),
		ParamTypeRef:  g.scope.GoFullTypeRef(source, sourcePkg),
		ResultTypeRef: g.scope.GoFullTypeRef(target, targetPkg),
		Code:          code,
	}
}

// IsCompatible returns an error if a and b are not both objects, both arrays,
// both maps or both the same primitive type. actx and bctx are used to build
// the error message if any.
func IsCompatible(a, b expr.DataType, actx, bctx string) error {
	switch {
	case expr.IsObject(a):
		if !expr.IsObject(b) {
			return fmt.Errorf("%s is an object but %s type is %s", actx, bctx, b.Name())
		}
	case expr.IsArray(a):
		if !expr.IsArray(b) {
			return fmt.Errorf("%s is an array but %s type is %s", actx, bctx, b.Name())
		}
	case expr.IsMap(a):
		if !expr.IsMap(b) {
			return fmt.Errorf("%s is a hash but %s type is %s", actx, bctx, b.Name())
		}
	default:
		if a.Kind() != b.Kind() {
			return fmt.Errorf("%s is a %s but %s type is %s", actx, a.Name(), bctx, b.Name())
		}
	}
	return nil
}

// AppendHelpers takes care of only appending helper functions from newH that
// are not already in oldH.
func AppendHelpers(oldH, newH []*TransformFunctionData) []*TransformFunctionData {
	for _, h := range newH {
		found := false
		for _, h2 := range oldH {
			if h.Name == h2.Name {
				found = true
				break
			}
		}
		if !found {
			oldH = append(oldH, h)
		}
	}
	return oldH
}

func (g *goTransformer) transformObject(source, target *expr.AttributeExpr, sourceVar, targetVar, sourcePkg, targetPkg string, newVar bool) (string, error) {
	buffer := &bytes.Buffer{}
	var (
		initCode     string
		postInitCode string
	)
	// iterate through attributes of primitive type first to initialize the
	// struct
	WalkMatches(source, target, func(src, tgt *expr.MappedAttributeExpr, srcAtt, _ *expr.AttributeExpr, n string) {
		if !expr.IsPrimitive(srcAtt.Type) {
			return
		}
		srcPtr := g.unmarshal || source.IsPrimitivePointer(n, !g.unmarshal)
		tgtPtr := target.IsPrimitivePointer(n, true)
		deref := ""
		srcField := sourceVar + "." + Goify(src.ElemName(n), true)
		if srcPtr && !tgtPtr {
			if !source.IsRequired(n) {
				postInitCode += fmt.Sprintf("if %s != nil {\n\t%s.%s = %s\n}\n",
					srcField, targetVar, Goify(tgt.ElemName(n), true), "*"+srcField)
				return
			}
			deref = "*"
		} else if !srcPtr && tgtPtr {
			deref = "&"
		}
		initCode += fmt.Sprintf("\n%s: %s%s,", Goify(tgt.ElemName(n), true), deref, srcField)
	})
	if initCode != "" {
		initCode += "\n"
	}
	assign := "="
	if newVar {
		assign = ":="
	}
	deref := "&"
	// if the target is a raw struct no need to return a pointer
	if _, ok := target.Type.(*expr.Object); ok {
		deref = ""
	}
	buffer.WriteString(fmt.Sprintf("%s %s %s%s{%s}\n", targetVar, assign, deref,
		g.scope.GoFullTypeName(target, targetPkg), initCode))
	buffer.WriteString(postInitCode)
	var err error
	WalkMatches(source, target, func(src, tgt *expr.MappedAttributeExpr, srcAtt, tgtAtt *expr.AttributeExpr, n string) {
		srcVar := sourceVar + "." + GoifyAtt(srcAtt, src.ElemName(n), true)
		tgtVar := targetVar + "." + GoifyAtt(tgtAtt, tgt.ElemName(n), true)
		err = IsCompatible(srcAtt.Type, tgtAtt.Type, srcVar, tgtVar)
		if err != nil {
			return
		}

		var (
			code string
		)
		_, ok := srcAtt.Type.(expr.UserType)
		switch {
		case expr.IsArray(srcAtt.Type):
			code, err = g.transformArray(expr.AsArray(srcAtt.Type), expr.AsArray(tgtAtt.Type), srcVar, tgtVar, sourcePkg, targetPkg, false)
		case expr.IsMap(srcAtt.Type):
			code, err = g.transformMap(expr.AsMap(srcAtt.Type), expr.AsMap(tgtAtt.Type), srcVar, tgtVar, sourcePkg, targetPkg, false)
		case ok:
			code = fmt.Sprintf("%s = %s(%s)\n", tgtVar, g.Helper(srcAtt, tgtAtt), srcVar)
		case expr.IsObject(srcAtt.Type):
			code, err = g.TransformAttribute(srcAtt, tgtAtt, srcVar, tgtVar, sourcePkg, targetPkg, false)
		}
		if err != nil {
			return
		}

		// We need to check for a nil source if it holds a reference
		// (pointer to primitive or an object, array or map) and is not
		// required. We also want to always check when unmarshaling if
		// the attribute type is not a primitive: either it's a user
		// type and we want to avoid calling transform helper functions
		// with nil value (if unmarshaling then requiredness has been
		// validated) or it's an object, map or array and we need to
		// check for nil to avoid making empty arrays and maps and to
		// avoid derefencing nil.
		var checkNil bool
		{
			isRef := !expr.IsPrimitive(srcAtt.Type) && !src.IsRequired(n) || src.IsPrimitivePointer(n, !g.unmarshal)
			marshalNonPrimitive := !g.unmarshal && !expr.IsPrimitive(srcAtt.Type)
			checkNil = isRef || marshalNonPrimitive
		}
		if code != "" && checkNil {
			code = fmt.Sprintf("if %s != nil {\n\t%s}\n", srcVar, code)
		}

		// Default value handling.
		//
		// There are 2 cases: one when generating marshaler code
		// (a.Unmarshal is false) and the other when generating
		// unmarshaler code (a.Unmarshal is true).
		//
		// When generating marshaler code we want to be lax and not
		// assume that required fields are set in case they have a
		// default value, instead the generated code is going to set the
		// fields to their default value (only applies to non-primitive
		// attributes).
		//
		// When generating unmarshaler code we rely on validations
		// running prior to this code so assume required fields are set.
		if tgt.HasDefaultValue(n) {
			if g.unmarshal {
				code += fmt.Sprintf("if %s == nil {\n\t", srcVar)
				if tgt.IsPrimitivePointer(n, true) {
					code += fmt.Sprintf("var tmp %s = %#v\n\t%s = &tmp\n", GoNativeTypeName(tgtAtt.Type), tgtAtt.DefaultValue, tgtVar)
				} else {
					code += fmt.Sprintf("%s = %#v\n", tgtVar, tgtAtt.DefaultValue)
				}
				code += "}\n"
			} else if src.IsPrimitivePointer(n, true) || !expr.IsPrimitive(srcAtt.Type) {
				code += fmt.Sprintf("if %s == nil {\n\t", srcVar)
				if tgt.IsPrimitivePointer(n, true) {
					code += fmt.Sprintf("var tmp %s = %#v\n\t%s = &tmp\n", GoNativeTypeName(tgtAtt.Type), tgtAtt.DefaultValue, tgtVar)
				} else {
					code += fmt.Sprintf("%s = %#v\n", tgtVar, tgtAtt.DefaultValue)
				}
				code += "}\n"
			}
		}

		buffer.WriteString(code)
	})
	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func (g *goTransformer) transformArray(source, target *expr.Array, sourceVar, targetVar, sourcePkg, targetPkg string, newVar bool) (string, error) {
	if err := IsCompatible(source.ElemType.Type, target.ElemType.Type, sourceVar+"[0]", targetVar+"[0]"); err != nil {
		return "", err
	}
	data := map[string]interface{}{
		"Source":      sourceVar,
		"Target":      targetVar,
		"TargetInit":  "",
		"NewVar":      newVar,
		"ElemTypeRef": g.scope.GoFullTypeRef(target.ElemType, targetPkg),
		"SourceElem":  source.ElemType,
		"TargetElem":  target.ElemType,
		"SourcePkg":   sourcePkg,
		"TargetPkg":   targetPkg,
		"Transformer": g,
		"LoopVar":     string(105 + strings.Count(targetVar, "[")),
	}
	return TransformArray(data), nil
}

// TransformArray transforms source array type to target array.
func TransformArray(data map[string]interface{}) string {
	var buf bytes.Buffer
	if err := transformArrayT.Execute(&buf, data); err != nil {
		panic(err) // bug
	}
	return buf.String()
}

func (g *goTransformer) transformMap(source, target *expr.Map, sourceVar, targetVar, sourcePkg, targetPkg string, newVar bool) (string, error) {
	if err := IsCompatible(source.KeyType.Type, target.KeyType.Type, sourceVar+".key", targetVar+".key"); err != nil {
		return "", err
	}
	if err := IsCompatible(source.ElemType.Type, target.ElemType.Type, sourceVar+"[*]", targetVar+"[*]"); err != nil {
		return "", err
	}
	data := map[string]interface{}{
		"Source":      sourceVar,
		"Target":      targetVar,
		"TargetInit":  "",
		"NewVar":      newVar,
		"KeyTypeRef":  g.scope.GoFullTypeRef(target.KeyType, targetPkg),
		"ElemTypeRef": g.scope.GoFullTypeRef(target.ElemType, targetPkg),
		"SourceKey":   source.KeyType,
		"TargetKey":   target.KeyType,
		"SourceElem":  source.ElemType,
		"TargetElem":  target.ElemType,
		"SourcePkg":   sourcePkg,
		"TargetPkg":   targetPkg,
		"Transformer": g,
		"LoopVar":     "",
	}
	return TransformMap(data, target), nil
}

// TransformMap transforms source map type to target map.
func TransformMap(data map[string]interface{}, target *expr.Map) string {
	if depth := mapDepth(target); depth > 0 {
		data["LoopVar"] = string(97 + depth)
	}
	var buf bytes.Buffer
	if err := transformMapT.Execute(&buf, data); err != nil {
		panic(err) // bug
	}
	return buf.String()
}

// mapDepth returns the level of nested maps. If map not nested, it returns 0.
func mapDepth(mp *expr.Map) int {
	return traverseMap(mp.ElemType.Type, 0)
}

func traverseMap(dt expr.DataType, depth int, seen ...map[string]struct{}) int {
	if mp := expr.AsMap(dt); mp != nil {
		depth++
		depth = traverseMap(mp.ElemType.Type, depth, seen...)
	} else if ar := expr.AsArray(dt); ar != nil {
		depth = traverseMap(ar.ElemType.Type, depth, seen...)
	} else if mo := expr.AsObject(dt); mo != nil {
		var s map[string]struct{}
		if len(seen) > 0 {
			s = seen[0]
		} else {
			s = make(map[string]struct{})
			seen = append(seen, s)
		}
		key := dt.Name()
		if u, ok := dt.(expr.UserType); ok {
			key = u.ID()
		}
		if _, ok := s[key]; ok {
			return depth
		}
		s[key] = struct{}{}
		var level int
		for _, nat := range *mo {
			// if object type has attributes of type map then find out the attribute that has
			// the deepest level of nested maps
			lvl := 0
			lvl = traverseMap(nat.Attribute.Type, lvl, seen...)
			if lvl > level {
				level = lvl
			}
		}
		depth += level
	}
	return depth
}

// TransformObjectHelpers collects the helper functions required to transform
// source object type to target object.
func TransformObjectHelpers(source, target expr.DataType, sourcePkg, targetPkg string, t Transformer, seen ...map[string]*TransformFunctionData) ([]*TransformFunctionData, error) {
	var (
		helpers []*TransformFunctionData
		err     error

		satt = &expr.AttributeExpr{Type: source}
		tatt = &expr.AttributeExpr{Type: target}
	)
	WalkMatches(satt, tatt, func(src, tgt *expr.MappedAttributeExpr, srcAtt, tgtAtt *expr.AttributeExpr, n string) {
		if err != nil {
			return
		}
		h, err2 := collectHelpers(srcAtt, tgtAtt, sourcePkg, targetPkg, src.IsRequired(n), t, seen...)
		if err2 != nil {
			err = err2
			return
		}
		helpers = append(helpers, h...)
	})
	if err != nil {
		return nil, err
	}
	return helpers, nil
}

// collectHelpers recursively traverses the given attributes and return the
// transform helper functions required to generate the transform code.
func collectHelpers(source, target *expr.AttributeExpr, sourcePkg, targetPkg string, req bool, t Transformer, seen ...map[string]*TransformFunctionData) ([]*TransformFunctionData, error) {
	var data []*TransformFunctionData
	switch {
	case expr.IsArray(source.Type):
		helpers, err := t.TransformAttributeHelpers(
			expr.AsArray(source.Type).ElemType.Type,
			expr.AsArray(target.Type).ElemType.Type,
			sourcePkg, targetPkg, seen...)
		if err != nil {
			return nil, err
		}
		data = append(data, helpers...)
	case expr.IsMap(source.Type):
		helpers, err := t.TransformAttributeHelpers(
			expr.AsMap(source.Type).KeyType.Type,
			expr.AsMap(target.Type).KeyType.Type,
			sourcePkg, targetPkg, seen...)
		if err != nil {
			return nil, err
		}
		data = append(data, helpers...)
		helpers, err = t.TransformAttributeHelpers(
			expr.AsMap(source.Type).ElemType.Type,
			expr.AsMap(target.Type).ElemType.Type,
			sourcePkg, targetPkg, seen...)
		if err != nil {
			return nil, err
		}
		data = append(data, helpers...)
	case expr.IsObject(source.Type):
		if ut, ok := source.Type.(expr.UserType); ok {
			name := t.Helper(source, target)
			var s map[string]*TransformFunctionData
			if len(seen) > 0 {
				s = seen[0]
			} else {
				s = make(map[string]*TransformFunctionData)
				seen = append(seen, s)
			}
			if _, ok := s[name]; ok {
				return nil, nil
			}
			code, err := t.TransformAttribute(ut.Attribute(), target,
				"v", "res", sourcePkg, targetPkg, true)
			if err != nil {
				return nil, err
			}
			if !req {
				code = "if v == nil {\n\treturn nil\n}\n" + code
			}
			tfd := t.TransformFunctionData(source, target, sourcePkg, targetPkg, code)
			s[name] = tfd
			data = append(data, tfd)
		}
		var err error
		WalkMatches(source, target, func(srcm, _ *expr.MappedAttributeExpr, src, tgt *expr.AttributeExpr, n string) {
			var helpers []*TransformFunctionData
			helpers, err = collectHelpers(src, tgt, sourcePkg, targetPkg, srcm.IsRequired(n), t, seen...)
			if err != nil {
				return
			}
			data = append(data, helpers...)
		})
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

// WalkMatches iterates through the source attribute expression and executes
// the walker function.
func WalkMatches(source, target *expr.AttributeExpr, walker func(src, tgt *expr.MappedAttributeExpr, srcc, tgtc *expr.AttributeExpr, n string)) {
	src := expr.NewMappedAttributeExpr(source)
	tgt := expr.NewMappedAttributeExpr(target)
	srcObj := expr.AsObject(src.Type)
	tgtObj := expr.AsObject(tgt.Type)
	// Map source object attribute names to target object attributes
	attributeMap := make(map[string]*expr.AttributeExpr)
	for _, nat := range *srcObj {
		if att := tgtObj.Attribute(nat.Name); att != nil {
			attributeMap[nat.Name] = att
		}
	}
	for _, natt := range *srcObj {
		n := natt.Name
		tgtc, ok := attributeMap[n]
		if !ok {
			continue
		}
		walker(src, tgt, natt.Attribute, tgtc, n)
	}
}

// used by template
func transformAttributeHelper(source, target *expr.AttributeExpr, sourceVar, targetVar, sourcePkg, targetPkg string, t Transformer, newVar bool) (string, error) {
	return t.TransformAttribute(source, target, sourceVar, targetVar, sourcePkg, targetPkg, newVar)
}

const transformArrayTmpl = `{{ if .TargetInit }}{{ .TargetInit }}{{ end -}}
{{ .Target }} {{ if .NewVar }}:{{ end }}= make([]{{ .ElemTypeRef }}, len({{ .Source }}))
for {{ .LoopVar }}, val := range {{ .Source }} {
	{{ transformAttribute .SourceElem .TargetElem "val" (printf "%s[%s]" .Target .LoopVar) .SourcePkg .TargetPkg .Transformer false -}}
}
`

const transformMapTmpl = `{{ if .TargetInit }}{{ .TargetInit }}{{ end -}}
{{ .Target }} {{ if .NewVar }}:{{ end }}= make(map[{{ .KeyTypeRef }}]{{ .ElemTypeRef }}, len({{ .Source }}))
for key, val := range {{ .Source }} {
	{{ transformAttribute .SourceKey .TargetKey "key" "tk" .SourcePkg .TargetPkg .Transformer true -}}
	{{ transformAttribute .SourceElem .TargetElem "val" (printf "tv%s" .LoopVar) .SourcePkg .TargetPkg .Transformer true -}}
	{{ .Target }}[tk] = {{ printf "tv%s" .LoopVar }}
}
`
