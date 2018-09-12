package codegen

import (
	"strconv"
	"testing"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

func TestProtoBufMessageDef(t *testing.T) {
	var (
		simpleArray = codegen.NewArray(expr.Boolean)
		nestedArray = codegen.NewArray(simpleArray.Type)
		simpleMap   = codegen.NewMap(expr.Int, expr.String)
		nestedMap   = codegen.NewMap(expr.Int, simpleMap.Type)
		ut          = &expr.UserTypeExpr{AttributeExpr: &expr.AttributeExpr{Type: expr.Boolean}, TypeName: "UserType"}
		obj         = objectRPC("IntField", expr.Int, "ArrayField", simpleArray.Type, "MapField", simpleMap.Type, "UserTypeField", ut)
		rt          = &expr.ResultTypeExpr{UserTypeExpr: &expr.UserTypeExpr{AttributeExpr: &expr.AttributeExpr{Type: expr.Boolean}, TypeName: "ResultType"}, Identifier: "application/vnd.goa.example", Views: nil}
		userType    = &expr.AttributeExpr{Type: ut}
		resultType  = &expr.AttributeExpr{Type: rt}
		threeDArray = codegen.NewArray(codegen.NewArray(codegen.NewArray(ut).Type).Type)

		nestedMapWithArray = codegen.NewMap(expr.Int, codegen.NewMap(expr.Int, codegen.NewArray(ut).Type).Type)
	)
	cases := map[string]struct {
		att      *expr.AttributeExpr
		expected string
	}{
		"BooleanKind": {&expr.AttributeExpr{Type: expr.Boolean}, "bool"},
		"IntKind":     {&expr.AttributeExpr{Type: expr.Int}, "sint32"},
		"Int32Kind":   {&expr.AttributeExpr{Type: expr.Int32}, "sint32"},
		"Int64Kind":   {&expr.AttributeExpr{Type: expr.Int64}, "sint64"},
		"UIntKind":    {&expr.AttributeExpr{Type: expr.UInt}, "uint32"},
		"UInt32Kind":  {&expr.AttributeExpr{Type: expr.UInt32}, "uint32"},
		"UInt64Kind":  {&expr.AttributeExpr{Type: expr.UInt64}, "uint64"},
		"Float32Kind": {&expr.AttributeExpr{Type: expr.Float32}, "float"},
		"Float64Kind": {&expr.AttributeExpr{Type: expr.Float64}, "double"},
		"StringKind":  {&expr.AttributeExpr{Type: expr.String}, "string"},
		"BytesKind":   {&expr.AttributeExpr{Type: expr.Bytes}, "bytes"},

		"Array":           {simpleArray, "repeated bool"},
		"ArrayOfArray":    {nestedArray, "repeated ArrayOfBool"},
		"3-D Array":       {threeDArray, "repeated ArrayOfArrayOfUserType"},
		"Map":             {simpleMap, "map<sint32, string>"},
		"MapOfMap":        {nestedMap, "map<sint32, MapOfSint32String>"},
		"MapOfMapOfArray": {nestedMapWithArray, "map<sint32, MapOfSint32ArrayOfUserType>"},
		"UserTypeExpr":    {userType, "UserType"},
		"ResultTypeExpr":  {resultType, "ResultType"},

		"Object": {obj, " {\n\tsint32 int_field = 1;\n\trepeated bool array_field = 2;\n\tmap<sint32, string> map_field = 3;\n\tUserType user_type_field = 4;\n}"},
	}

	for k, tc := range cases {
		scope := codegen.NewNameScope()
		actual := protoBufMessageDef(tc.att, scope)
		if actual != tc.expected {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func objectRPC(params ...interface{}) *expr.AttributeExpr {
	obj := expr.Object{}
	for i := 0; i < len(params); i += 2 {
		name := params[i].(string)
		typ := params[i+1].(expr.DataType)
		obj = append(obj, &expr.NamedAttributeExpr{
			Name: name,
			Attribute: &expr.AttributeExpr{
				Type: typ,
				Meta: expr.MetaExpr{"rpc:tag": []string{strconv.Itoa(int(i/2) + 1)}},
			},
		})
	}
	return &expr.AttributeExpr{Type: &obj}
}
