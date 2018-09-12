package codegen

import (
	"testing"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

var (
	arrayUInt   = codegen.NewArray(expr.UInt)
	arrayUT     = codegen.NewArray(userType)
	arrayMap    = codegen.NewArray(mapStringUT.Type)
	nestedArray = codegen.NewArray(codegen.NewArray(userType).Type)

	mapUIntInt     = codegen.NewMap(expr.UInt, expr.Int)
	mapStringUT    = codegen.NewMap(expr.String, userType)
	mapStringArray = codegen.NewMap(expr.String, nestedArray.Type)
	nestedMap      = codegen.NewMap(expr.UInt, mapStringUT.Type)

	objNoRequiredNoDefault = codegen.NewObject("a", expr.String, "b", expr.Int)
	objRequired            = codegen.SetRequired(expr.DupAtt(objNoRequiredNoDefault), "a", "b")
	objDefault             = codegen.SetDefault(expr.DupAtt(objNoRequiredNoDefault), "a", "foo", "b", "1")
	objMixed               = codegen.SetRequired(codegen.NewObject("String", expr.String, "Int", expr.Int, "Array", arrayUInt.Type, "Map", mapUIntInt.Type, "UT", userType), "String", "Array", "UT")
	objWithArrayMap        = codegen.NewObject("a", arrayUT.Type, "b", mapStringUT.Type)

	userType = &expr.UserTypeExpr{TypeName: "UserType", AttributeExpr: objRequired}
	mixedUT  = &expr.UserTypeExpr{TypeName: "mixedUserType", AttributeExpr: objMixed}
)

func TestProtoBufTypeTransform(t *testing.T) {
	var (
		sourceVar = "source"
		targetVar = "target"
	)
	cases := []struct {
		Name           string
		Source, Target *expr.AttributeExpr
		ToProto        bool
		TargetPkg      string

		Code string
	}{
		// test cases to transform goa type to protocol buffer type
		{"obj-no-required-no-default-to-protobuf", objNoRequiredNoDefault, objNoRequiredNoDefault, true, "", objNoRequiredNoDefaultToProtoCode},
		{"obj-required-to-protobuf", objRequired, objRequired, true, "", objRequiredToProtoCode},
		{"obj-default-to-protobuf", objDefault, objDefault, true, "", objDefaultToProtoCode},
		{"obj-with-array-map-to-protobuf", objWithArrayMap, objWithArrayMap, true, "", objWithArrayMapToProtoCode},
		{"obj-mixed-to-protobuf", objMixed, objMixed, true, "", objMixedToProtoCode},
		{"array-of-uint-to-protobuf", arrayUInt, arrayUInt, true, "", arrayUIntToProtoCode},
		{"map-of-uint-int-to-protobuf", mapUIntInt, mapUIntInt, true, "", mapUIntIntToProtoCode},
		{"map-of-string-array-to-protobuf", mapStringArray, mapStringArray, true, "", mapStringArrayToProtoCode},
		{"nested-array-to-protobuf", nestedArray, nestedArray, true, "", nestedArrayToProtoCode},
		{"array-of map-to-protobuf", arrayMap, arrayMap, true, "", arrayOfMapToProtoCode},
		{"nested-map-to-protobuf", nestedMap, nestedMap, true, "", nestedMapToProtoCode},

		// test cases to transform protocol buffer type to goa type
		{"obj-no-required-no-default-to-goa", objNoRequiredNoDefault, objNoRequiredNoDefault, false, "", objNoRequiredNoDefaultToGoaCode},
		{"obj-required-to-goa", objRequired, objRequired, false, "", objRequiredToGoaCode},
		{"obj-default-to-goa", objDefault, objDefault, false, "", objDefaultToGoaCode},
		{"obj-with-array-map-to-goa", objWithArrayMap, objWithArrayMap, false, "", objWithArrayMapToGoaCode},
		{"obj-mixed-to-protobuf", objMixed, objMixed, false, "", objMixedToGoaCode},
		{"array-of-uint-to-goa", arrayUInt, arrayUInt, false, "", arrayUIntToGoaCode},
		{"map-of-uint-int-to-goa", mapUIntInt, mapUIntInt, false, "", mapUIntIntToGoaCode},
		{"map-of-string-array-to-goa", mapStringArray, mapStringArray, false, "", mapStringArrayToGoaCode},
		{"nested-array-to-goa", nestedArray, nestedArray, false, "", nestedArrayToGoaCode},
		{"array-of map-to-goa", arrayMap, arrayMap, false, "", arrayOfMapToGoaCode},
		{"nested-map-to-goa", nestedMap, nestedMap, false, "", nestedMapToGoaCode},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			src := &expr.UserTypeExpr{TypeName: "SourceType", AttributeExpr: c.Source}
			tgt := &expr.UserTypeExpr{TypeName: "TargetType", AttributeExpr: c.Target}
			code, _, err := protoBufTypeTransform(src, tgt, sourceVar, targetVar, "", c.TargetPkg, c.ToProto, codegen.NewNameScope())
			if err != nil {
				t.Fatal(err)
			}
			code = codegen.FormatTestCode(t, "package foo\nfunc transform(){\n"+code+"}")
			if code != c.Code {
				t.Errorf("invalid code, got:\n%s\ngot vs. expected:\n%s", code, codegen.Diff(t, code, c.Code))
			}
		})
	}
}

const (
	objNoRequiredNoDefaultToProtoCode = `func transform() {
	target := &TargetType{}
	if source.A != nil {
		target.A = *source.A
	}
	if source.B != nil {
		target.B = int32(*source.B)
	}
}
`

	objNoRequiredNoDefaultToGoaCode = `func transform() {
	target := &TargetType{
		A: &source.A,
	}
	bptr := int(source.B)
	target.B = &bptr
}
`

	objRequiredToProtoCode = `func transform() {
	target := &TargetType{
		A: source.A,
		B: int32(source.B),
	}
}
`

	objRequiredToGoaCode = `func transform() {
	target := &TargetType{
		A: source.A,
		B: int(source.B),
	}
}
`

	objDefaultToProtoCode = `func transform() {
	target := &TargetType{
		A: source.A,
		B: int32(source.B),
	}
}
`

	objDefaultToGoaCode = `func transform() {
	target := &TargetType{
		A: source.A,
		B: int(source.B),
	}
}
`

	objWithArrayMapToProtoCode = `func transform() {
	target := &TargetType{}
	if source.A != nil {
		target.A = make([]*UserType, len(source.A))
		for i, val := range source.A {
			target.A[i] = &UserType{
				A: val.A,
				B: int32(val.B),
			}
		}
	}
	if source.B != nil {
		target.B = make(map[string]*UserType, len(source.B))
		for key, val := range source.B {
			tk := key
			tv := &UserType{
				A: val.A,
				B: int32(val.B),
			}
			target.B[tk] = tv
		}
	}
}
`

	objWithArrayMapToGoaCode = `func transform() {
	target := &TargetType{}
	if source.A != nil {
		target.A = make([]*UserType, len(source.A))
		for i, val := range source.A {
			target.A[i] = &UserType{
				A: val.A,
				B: int(val.B),
			}
		}
	}
	if source.B != nil {
		target.B = make(map[string]*UserType, len(source.B))
		for key, val := range source.B {
			tk := key
			tv := &UserType{
				A: val.A,
				B: int(val.B),
			}
			target.B[tk] = tv
		}
	}
}
`

	objMixedToProtoCode = `func transform() {
	target := &TargetType{
		String: source.String,
	}
	if source.Int != nil {
		target.Int = int32(*source.Int)
	}
	target.Array = make([]uint32, len(source.Array))
	for i, val := range source.Array {
		target.Array[i] = uint32(val)
	}
	if source.Map != nil {
		target.Map = make(map[uint32]int32, len(source.Map))
		for key, val := range source.Map {
			tk := uint32(key)
			tv := int32(val)
			target.Map[tk] = tv
		}
	}
	target.UT = userTypeToUserTypeProtoBuf(source.UT)
}
`

	objMixedToGoaCode = `func transform() {
	target := &TargetType{
		String: source.String,
	}
	int_ptr := int(source.Int)
	target.Int = &int_ptr
	target.Array = make([]uint, len(source.Array))
	for i, val := range source.Array {
		target.Array[i] = uint(val)
	}
	if source.Map != nil {
		target.Map = make(map[uint]int, len(source.Map))
		for key, val := range source.Map {
			tk := uint(key)
			tv := int(val)
			target.Map[tk] = tv
		}
	}
	target.UT = userTypeProtoBufToUserType(source.UT)
}
`

	arrayUIntToProtoCode = `func transform() {
	target := make([]uint32, len(source))
	for i, val := range source {
		target[i] = uint32(val)
	}
}
`

	arrayUIntToGoaCode = `func transform() {
	target := make([]uint, len(source))
	for i, val := range source {
		target[i] = uint(val)
	}
}
`

	nestedArrayToProtoCode = `func transform() {
	target := make([]*ArrayOfUserType, len(source))
	for i, val := range source {
		target[i].Field = make([]*UserType, len(val))
		for j, val := range val {
			target[i].Field[j] = &UserType{
				A: val.A,
				B: int32(val.B),
			}
		}
	}
}
`

	nestedArrayToGoaCode = `func transform() {
	target := make([][]*UserType, len(source))
	for i, val := range source {
		target[i] = make([]*UserType, len(val.Field))
		for j, val := range val.Field {
			target[i][j] = &UserType{
				A: val.A,
				B: int(val.B),
			}
		}
	}
}
`

	arrayOfMapToProtoCode = `func transform() {
	target := make([]*MapOfStringUserType, len(source))
	for i, val := range source {
		target[i].Field = make(map[string]*UserType, len(val))
		for key, val := range val {
			tk := key
			tv := &UserType{
				A: val.A,
				B: int32(val.B),
			}
			target[i].Field[tk] = tv
		}
	}
}
`

	arrayOfMapToGoaCode = `func transform() {
	target := make([]map[string]*UserType, len(source))
	for i, val := range source {
		target[i] = make(map[string]*UserType, len(val.Field))
		for key, val := range val.Field {
			tk := key
			tv := &UserType{
				A: val.A,
				B: int(val.B),
			}
			target[i][tk] = tv
		}
	}
}
`

	mapUIntIntToProtoCode = `func transform() {
	target := make(map[uint32]int32, len(source))
	for key, val := range source {
		tk := uint32(key)
		tv := int32(val)
		target[tk] = tv
	}
}
`

	mapUIntIntToGoaCode = `func transform() {
	target := make(map[uint]int, len(source))
	for key, val := range source {
		tk := uint(key)
		tv := int(val)
		target[tk] = tv
	}
}
`

	mapStringArrayToProtoCode = `func transform() {
	target := make(map[string]*ArrayOfArrayOfUserType, len(source))
	for key, val := range source {
		tk := key
		tv := make([]*ArrayOfUserType, len(val))
		for i, val := range val {
			tv[i].Field = make([]*UserType, len(val))
			for j, val := range val {
				tv[i].Field[j] = &UserType{
					A: val.A,
					B: int32(val.B),
				}
			}
		}
		target[tk] = &ArrayOfArrayOfUserType{Field: tv}
	}
}
`

	mapStringArrayToGoaCode = `func transform() {
	target := make(map[string][][]*UserType, len(source))
	for key, val := range source {
		tk := key
		tv := make([][]*UserType, len(val.Field))
		for i, val := range val.Field {
			tv[i] = make([]*UserType, len(val.Field))
			for j, val := range val.Field {
				tv[i][j] = &UserType{
					A: val.A,
					B: int(val.B),
				}
			}
		}
		target[tk] = tv
	}
}
`

	nestedMapToProtoCode = `func transform() {
	target := make(map[uint32]*MapOfStringUserType, len(source))
	for key, val := range source {
		tk := uint32(key)
		tvb := make(map[string]*UserType, len(val))
		for key, val := range val {
			tk := key
			tv := &UserType{
				A: val.A,
				B: int32(val.B),
			}
			tvb[tk] = tv
		}
		target[tk] = &MapOfStringUserType{Field: tvb}
	}
}
`

	nestedMapToGoaCode = `func transform() {
	target := make(map[uint]map[string]*UserType, len(source))
	for key, val := range source {
		tk := uint(key)
		tvb := make(map[string]*UserType, len(val.Field))
		for key, val := range val.Field {
			tk := key
			tv := &UserType{
				A: val.A,
				B: int(val.B),
			}
			tvb[tk] = tv
		}
		target[tk] = tvb
	}
}
`
)
