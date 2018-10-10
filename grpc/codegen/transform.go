package codegen

import (
	"bytes"
	"fmt"
	"strings"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// protoTransformer implements the codegen.Transformer interface.
type protoTransformer struct {
	// proto if true indicates that the target type is a protocol buffer Go type.
	proto bool
	scope *codegen.NameScope
}

// protoBufTypeTransform produces Go code that initializes the data structure
// defined by target from an instance of the data structure described the
// source. Either the source or target is a type referring to the protocol
// buffer message type. The algorithm matches object fields by name and ignores
// object fields in target that don't have a match in source. The matching and
// generated code leverage mapped attributes so that attribute names may use
// the "name:elem" syntax to define the name of the design attribute and the
// name of the corresponding generated Go struct field. The function returns
// an error if target is not compatible with source (different type, fields of
// different type etc).
//
// sourceVar and targetVar contain the name of the variables that hold the
// source and target data structures respectively.
//
// sourcePkg and targetPkg contain the name of the Go package that defines the
// source or target type respectively in case it's not the same package as where
// the generated code lives.
//
// proto if true indicates whether the code is being generated to initialize
// a Go struct generated from the protocol buffer message type, otherwise to
// initialize a type from a Go struct generated from the protocol buffer message
// type.
//
//   - proto3 syntax is used to refer to a protocol buffer generated Go struct.
//
// scope is used to compute the name of the user types when initializing fields
// that use them.
//
func protoBufTypeTransform(source, target expr.DataType, sourceVar, targetVar, sourcePkg, targetPkg string, proto bool, scope *codegen.NameScope) (string, []*codegen.TransformFunctionData, error) {
	var (
		satt = &expr.AttributeExpr{Type: source}
		tatt = &expr.AttributeExpr{Type: target}
	)

	p := &protoTransformer{proto: proto, scope: scope}

	code, err := p.TransformAttribute(satt, tatt, sourceVar, targetVar, sourcePkg, targetPkg, true)
	if err != nil {
		return "", nil, err
	}

	funcs, err := p.TransformAttributeHelpers(source, target, sourcePkg, targetPkg)
	if err != nil {
		return "", nil, err
	}

	return strings.TrimRight(code, "\n"), funcs, nil
}

// TransformAttribute converts source attribute expression to target returning
// the conversion code and error (if any). Either source or target is a
// protocol buffer message type.
func (p *protoTransformer) TransformAttribute(source, target *expr.AttributeExpr, sourceVar, targetVar, sourcePkg, targetPkg string, newVar bool) (string, error) {
	var (
		code string
		err  error
	)
	{
		svcAtt := target
		if p.proto {
			svcAtt = source
		}
		switch {
		case expr.IsArray(svcAtt.Type):
			code, err = p.transformArray(source, target, sourceVar, targetVar, sourcePkg, targetPkg, newVar)
		case expr.IsMap(svcAtt.Type):
			code, err = p.transformMap(source, target, sourceVar, targetVar, sourcePkg, targetPkg, newVar)
		case expr.IsObject(svcAtt.Type):
			code, err = p.transformObject(source, target, sourceVar, targetVar, sourcePkg, targetPkg, newVar)
		default:
			code, err = p.transformPrimitive(source, target, sourceVar, targetVar, sourcePkg, targetPkg, newVar)
		}
	}
	if err != nil {
		return "", err
	}
	return code, nil
}

func (p *protoTransformer) TransformAttributeHelpers(source, target expr.DataType, sourcePkg, targetPkg string, seen ...map[string]*codegen.TransformFunctionData) ([]*codegen.TransformFunctionData, error) {
	if err := codegen.IsCompatible(source, target, "p", "res"); err != nil {
		if p.proto {
			target = unwrapAttr(&expr.AttributeExpr{Type: target}).Type
		} else {
			source = unwrapAttr(&expr.AttributeExpr{Type: source}).Type
		}
		if err = codegen.IsCompatible(source, target, "p", "res"); err != nil {
			return nil, err
		}
	}
	var (
		helpers []*codegen.TransformFunctionData
		err     error
	)
	// Do not generate a transform function for the top most user type.
	switch {
	case expr.IsArray(source):
		source = expr.AsArray(source).ElemType.Type
		target = expr.AsArray(target).ElemType.Type
		helpers, err = p.TransformAttributeHelpers(source, target, sourcePkg, targetPkg, seen...)
	case expr.IsMap(source):
		sm := expr.AsMap(source)
		tm := expr.AsMap(target)
		source = sm.ElemType.Type
		target = tm.ElemType.Type
		helpers, err = p.TransformAttributeHelpers(source, target, sourcePkg, targetPkg, seen...)
		if err == nil {
			var other []*codegen.TransformFunctionData
			source = sm.KeyType.Type
			target = tm.KeyType.Type
			other, err = p.TransformAttributeHelpers(source, target, sourcePkg, targetPkg, seen...)
			helpers = append(helpers, other...)
		}
	case expr.IsObject(source):
		helpers, err = codegen.TransformObjectHelpers(source, target, sourcePkg, targetPkg, p, seen...)
	}
	if err != nil {
		return nil, err
	}
	return helpers, nil
}

func (p *protoTransformer) TransformFunctionData(source, target *expr.AttributeExpr, sourcePkg, targetPkg, code string) *codegen.TransformFunctionData {
	var pref, resref string
	{
		if p.proto {
			pref = p.scope.GoFullTypeRef(source, sourcePkg)
			resref = protoBufGoFullTypeRef(target, targetPkg, p.scope)
		} else {
			pref = protoBufGoFullTypeRef(source, sourcePkg, p.scope)
			resref = p.scope.GoFullTypeRef(target, targetPkg)
		}
	}
	return &codegen.TransformFunctionData{
		Name:          p.Helper(source, target),
		ParamTypeRef:  pref,
		ResultTypeRef: resref,
		Code:          code,
	}
}

func (p *protoTransformer) Helper(src, tgt *expr.AttributeExpr) string {
	var (
		sname string
		tname string

		suffix = "ProtoBuf"
	)
	{
		sname = p.scope.GoTypeName(src)
		if _, ok := src.Meta["goa.external"]; ok {
			// type belongs to external package so name could clash
			sname += "Ext"
		}
		tname = p.scope.GoTypeName(tgt)
		if _, ok := tgt.Meta["goa.external"]; ok {
			// type belongs to external package so name could clash
			tname += "Ext"
		}
		if p.proto {
			tname += suffix
		} else {
			sname += suffix
		}
	}
	return codegen.Goify(sname+"To"+tname, false)
}

func (p *protoTransformer) transformPrimitive(source, target *expr.AttributeExpr, sourceVar, targetVar, sourcePkg, targetPkg string, newVar bool) (string, error) {
	var code string
	if err := codegen.IsCompatible(source.Type, target.Type, sourceVar, targetVar); err != nil {
		if p.proto {
			code += fmt.Sprintf("%s := &%s{}\n", targetVar, protoBufGoFullTypeName(target, targetPkg, p.scope))
			targetVar += ".Field"
			newVar = false
			target = unwrapAttr(target)
		} else {
			source = unwrapAttr(source)
			sourceVar += ".Field"
		}
		if err = codegen.IsCompatible(source.Type, target.Type, sourceVar, targetVar); err != nil {
			return "", err
		}
	}
	assign := "="
	if newVar {
		assign = ":="
	}
	code += fmt.Sprintf("%s %s %s\n", targetVar, assign, typeConvert(sourceVar, source.Type, target.Type, p.proto))
	return code, nil
}

func (p *protoTransformer) transformObject(source, target *expr.AttributeExpr, sourceVar, targetVar, sourcePkg, targetPkg string, newVar bool) (string, error) {
	if err := codegen.IsCompatible(source.Type, target.Type, sourceVar, targetVar); err != nil {
		if p.proto {
			target = unwrapAttr(target)
		} else {
			source = unwrapAttr(source)
		}
		if err = codegen.IsCompatible(source.Type, target.Type, sourceVar, targetVar); err != nil {
			return "", err
		}
	}
	var (
		initCode     string
		postInitCode string

		buffer = &bytes.Buffer{}
	)
	{
		// iterate through attributes of primitive type first to initialize the
		// struct
		codegen.WalkMatches(source, target, func(src, tgt *expr.MappedAttributeExpr, srcAtt, tgtAtt *expr.AttributeExpr, n string) {
			if !expr.IsPrimitive(srcAtt.Type) {
				return
			}
			var (
				srcFldName, tgtFldName string
				srcPtr, tgtPtr         bool
			)
			{
				if p.proto {
					srcPtr = source.IsPrimitivePointer(n, true)
					srcFldName = codegen.Goify(src.ElemName(n), true)
					// Protocol buffer does not care about common initialisms like
					// api -> API.
					tgtFldName = protoBufify(tgt.ElemName(n), true)
				} else {
					srcFldName = protoBufify(src.ElemName(n), true)
					tgtFldName = codegen.Goify(tgt.ElemName(n), true)
					tgtPtr = target.IsPrimitivePointer(n, true)
				}
			}
			deref := ""
			srcField := sourceVar + "." + srcFldName
			switch {
			case srcPtr && !tgtPtr:
				if !source.IsRequired(n) {
					postInitCode += fmt.Sprintf("if %s != nil {\n\t%s.%s = %s\n}\n",
						srcField, targetVar, tgtFldName, typeConvert("*"+srcField, srcAtt.Type, tgtAtt.Type, p.proto))
					return
				}
				deref = "*"
			case !srcPtr && tgtPtr:
				deref = "&"
				if sVar := typeConvert(srcField, srcAtt.Type, tgtAtt.Type, p.proto); sVar != srcField {
					// type cast is required
					tgtName := codegen.Goify(tgt.ElemName(n), false)
					postInitCode += fmt.Sprintf("%sptr := %s\n%s.%s = %s%sptr\n", tgtName, sVar, targetVar, tgtFldName, deref, tgtName)
					return
				}
			}
			initCode += fmt.Sprintf("\n%s: %s%s,", tgtFldName, deref, typeConvert(srcField, srcAtt.Type, tgtAtt.Type, p.proto))
		})
	}
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
		p.scope.GoFullTypeName(target, targetPkg), initCode))
	buffer.WriteString(postInitCode)

	var err error
	{
		codegen.WalkMatches(source, target, func(src, tgt *expr.MappedAttributeExpr, srcAtt, tgtAtt *expr.AttributeExpr, n string) {
			var srcFldName, tgtFldName string
			{
				if p.proto {
					srcFldName = codegen.GoifyAtt(srcAtt, src.ElemName(n), true)
					tgtFldName = protoBufifyAtt(tgtAtt, tgt.ElemName(n), true)
				} else {
					srcFldName = protoBufifyAtt(tgtAtt, tgt.ElemName(n), true)
					tgtFldName = codegen.GoifyAtt(srcAtt, src.ElemName(n), true)
				}
			}
			srcVar := sourceVar + "." + srcFldName
			tgtVar := targetVar + "." + tgtFldName
			err = codegen.IsCompatible(srcAtt.Type, tgtAtt.Type, srcVar, tgtVar)
			if err != nil {
				return
			}

			var (
				code string
			)
			{
				_, ok := srcAtt.Type.(expr.UserType)
				switch {
				case expr.IsArray(srcAtt.Type):
					code, err = p.transformArray(srcAtt, tgtAtt, srcVar, tgtVar, sourcePkg, targetPkg, false)
				case expr.IsMap(srcAtt.Type):
					code, err = p.transformMap(srcAtt, tgtAtt, srcVar, tgtVar, sourcePkg, targetPkg, false)
				case ok:
					code = fmt.Sprintf("%s = %s(%s)\n", tgtVar, p.Helper(srcAtt, tgtAtt), srcVar)
				case expr.IsObject(srcAtt.Type):
					code, err = p.TransformAttribute(srcAtt, tgtAtt, srcVar, tgtVar, sourcePkg, targetPkg, false)
				}
				if err != nil {
					return
				}

				// Nil check handling.
				//
				// We need to check for a nil source if it holds a reference
				// (pointer to primitive or an object, array or map) and is not
				// required. If source is a protocol buffer generated Go type,
				// the attributes of primitive type are always non-pointers (even if
				// not required). We don't have to check for nil in that case.
				var checkNil bool
				{
					checkNil = !expr.IsPrimitive(srcAtt.Type) && !src.IsRequired(n) || src.IsPrimitivePointer(n, true) && !p.proto
				}
				if code != "" && checkNil {
					code = fmt.Sprintf("if %s != nil {\n\t%s}\n", srcVar, code)
				}

				// Default value handling.
				// proto3 does not support non-zero default values. It is impossible to
				// find out from the protocol buffer type whether the primitive fields
				// (always non-pointers) are set to zero values as default or by the
				// application itself.
				if tgt.HasDefaultValue(n) {
					if p.proto {
						if src.IsPrimitivePointer(n, true) || !expr.IsPrimitive(srcAtt.Type) {
							code += fmt.Sprintf("if %s == nil {\n\t", srcVar)
							code += fmt.Sprintf("%s = %#v\n", tgtVar, tgtAtt.DefaultValue)
							code += "}\n"
						}
					} else if !expr.IsPrimitive(srcAtt.Type) {
						code += fmt.Sprintf("if %s == nil {\n\t", srcVar)
						code += fmt.Sprintf("%s = %#v\n", tgtVar, tgtAtt.DefaultValue)
						code += "}\n"
					}
				}
			}
			buffer.WriteString(code)
		})
	}
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (p *protoTransformer) transformArray(source, target *expr.AttributeExpr, sourceVar, targetVar, sourcePkg, targetPkg string, newVar bool) (string, error) {
	var (
		src, tgt *expr.Array
		tgtInit  string
	)
	{
		src = expr.AsArray(source.Type)
		tgt = expr.AsArray(target.Type)
		if err := codegen.IsCompatible(source.Type, target.Type, sourceVar+"[0]", targetVar+"[0]"); err != nil {
			if p.proto {
				tgt = expr.AsArray(unwrapAttr(target).Type)
				assign := "="
				if newVar {
					assign = ":="
				}
				tgtInit = fmt.Sprintf("%s %s &%s{}\n", targetVar, assign, protoBufGoFullTypeName(target, targetPkg, p.scope))
				targetVar += ".Field"
				newVar = false
			} else {
				src = expr.AsArray(unwrapAttr(source).Type)
				sourceVar += ".Field"
			}
			if _, err := p.TransformAttribute(src.ElemType, tgt.ElemType, sourceVar, targetVar, sourcePkg, targetPkg, newVar); err != nil {
				return "", err
			}
		}
	}
	var (
		elemRef string
	)
	{
		if p.proto {
			elemRef = protoBufGoFullTypeRef(tgt.ElemType, targetPkg, p.scope)
		} else {
			elemRef = p.scope.GoFullTypeRef(tgt.ElemType, targetPkg)
		}
	}
	data := map[string]interface{}{
		"Source":      sourceVar,
		"Target":      targetVar,
		"TargetInit":  tgtInit,
		"NewVar":      newVar,
		"ElemTypeRef": elemRef,
		"SourceElem":  src.ElemType,
		"TargetElem":  tgt.ElemType,
		"SourcePkg":   sourcePkg,
		"TargetPkg":   targetPkg,
		"Transformer": p,
		"LoopVar":     string(105 + strings.Count(targetVar, "[")),
	}
	return codegen.TransformArray(data), nil
}

func (p *protoTransformer) transformMap(source, target *expr.AttributeExpr, sourceVar, targetVar, sourcePkg, targetPkg string, newVar bool) (string, error) {
	var (
		src, tgt *expr.Map
		tgtInit  string
	)
	{
		src = expr.AsMap(source.Type)
		tgt = expr.AsMap(target.Type)
		if err := codegen.IsCompatible(source.Type, target.Type, sourceVar+"[*]", targetVar+"[*]"); err != nil {
			if p.proto {
				tgt = expr.AsMap(unwrapAttr(target).Type)
				assign := "="
				if newVar {
					assign = ":="
				}
				tgtInit = fmt.Sprintf("%s %s &%s{}\n", targetVar, assign, protoBufGoFullTypeName(target, targetPkg, p.scope))
				targetVar += ".Field"
				newVar = false
			} else {
				src = expr.AsMap(unwrapAttr(source).Type)
				sourceVar += ".Field"
			}
			if _, err := p.TransformAttribute(src.ElemType, tgt.ElemType, sourceVar, targetVar, sourcePkg, targetPkg, newVar); err != nil {
				return "", err
			}
		}
	}
	if err := codegen.IsCompatible(src.KeyType.Type, tgt.KeyType.Type, sourceVar+".key", targetVar+".key"); err != nil {
		return "", err
	}
	var (
		keyRef, elemRef string
	)
	{
		if p.proto {
			keyRef = protoBufGoFullTypeRef(tgt.KeyType, targetPkg, p.scope)
			elemRef = protoBufGoFullTypeRef(tgt.ElemType, targetPkg, p.scope)
		} else {
			keyRef = p.scope.GoFullTypeRef(tgt.KeyType, targetPkg)
			elemRef = p.scope.GoFullTypeRef(tgt.ElemType, targetPkg)
		}
	}
	data := map[string]interface{}{
		"Source":      sourceVar,
		"Target":      targetVar,
		"TargetInit":  tgtInit,
		"NewVar":      newVar,
		"KeyTypeRef":  keyRef,
		"ElemTypeRef": elemRef,
		"SourceKey":   src.KeyType,
		"TargetKey":   tgt.KeyType,
		"SourceElem":  src.ElemType,
		"TargetElem":  tgt.ElemType,
		"SourcePkg":   sourcePkg,
		"TargetPkg":   targetPkg,
		"Transformer": p,
		"LoopVar":     "",
	}
	return codegen.TransformMap(data, tgt), nil
}

// typeConvert converts the source attribute type based on the target type.
// NOTE: For Int and UInt kinds, protocol buffer Go compiler generates
// int32 and uint32 respectively whereas goa v2 generates int and uint.
//
// proto if true indicates that the target attribute is a protocol buffer type.
func typeConvert(sourceVar string, source, target expr.DataType, proto bool) string {
	if source.Kind() != expr.IntKind && source.Kind() != expr.UIntKind {
		return sourceVar
	}
	if proto {
		sourceVar = fmt.Sprintf("%s(%s)", protoBufNativeGoTypeName(source), sourceVar)
	} else {
		sourceVar = fmt.Sprintf("%s(%s)", codegen.GoNativeTypeName(source), sourceVar)
	}
	return sourceVar
}
