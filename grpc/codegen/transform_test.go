package codegen

import (
	"testing"

	"goa.design/goa/codegen"
	"goa.design/goa/design"
)

var (
	arrayUInt = codegen.NewArray(design.UInt)
	arrayUT   = codegen.NewArray(userType)

	mapUIntInt  = codegen.NewMap(design.UInt, design.Int)
	mapStringUT = codegen.NewMap(design.String, userType)

	objNoRequiredNoDefault = codegen.NewObject("a", design.String, "b", design.Int)
	objRequired            = codegen.SetRequired(design.DupAtt(objNoRequiredNoDefault), "a", "b")
	objDefault             = codegen.SetDefault(design.DupAtt(objNoRequiredNoDefault), "a", "foo", "b", "1")
	objMixed               = codegen.SetRequired(codegen.NewObject("String", design.String, "Int", design.Int, "Array", arrayUInt.Type, "Map", mapUIntInt.Type, "UT", userType), "String", "Array", "UT")
	objWithArrayMap        = codegen.NewObject("a", arrayUT.Type, "b", mapStringUT.Type)

	userType = &design.UserTypeExpr{TypeName: "UserType", AttributeExpr: objRequired}
	mixedUT  = &design.UserTypeExpr{TypeName: "mixedUserType", AttributeExpr: objMixed}
)

func TestProtoBufTypeTransform(t *testing.T) {
	var (
		sourceVar = "source"
		targetVar = "target"
	)
	cases := []struct {
		Name           string
		Source, Target *design.AttributeExpr
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

		// test cases to transform protocol buffer type to goa type
		{"obj-no-required-no-default-to-goa", objNoRequiredNoDefault, objNoRequiredNoDefault, false, "", objNoRequiredNoDefaultToGoaCode},
		{"obj-required-to-goa", objRequired, objRequired, false, "", objRequiredToGoaCode},
		{"obj-default-to-goa", objDefault, objDefault, false, "", objDefaultToGoaCode},
		{"obj-with-array-map-to-goa", objWithArrayMap, objWithArrayMap, false, "", objWithArrayMapToGoaCode},
		{"obj-mixed-to-protobuf", objMixed, objMixed, false, "", objMixedToGoaCode},
		{"array-of-uint-to-goa", arrayUInt, arrayUInt, false, "", arrayUIntToGoaCode},
		{"map-of-uint-int-to-goa", mapUIntInt, mapUIntInt, false, "", mapUIntIntToGoaCode},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			src := &design.UserTypeExpr{TypeName: "SourceType", AttributeExpr: c.Source}
			tgt := &design.UserTypeExpr{TypeName: "TargetType", AttributeExpr: c.Target}
			code, _, err := ProtoBufTypeTransform(src, tgt, sourceVar, targetVar, "", c.TargetPkg, c.ToProto, codegen.NewNameScope())
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
	target.Array = make([]uint, len(source.Array))
	for i, val := range source.Array {
		target.Array[i] = uint32(val)
	}
	if source.Map != nil {
		target.Map = make(map[uint]int, len(source.Map))
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
	target := make([]uint, len(source))
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

	mapUIntIntToProtoCode = `func transform() {
	target := make(map[uint]int, len(source))
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
)
