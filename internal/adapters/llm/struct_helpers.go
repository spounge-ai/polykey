package llm


import (
    "google.golang.org/protobuf/types/known/structpb"
)


// dev: Using float64 since its conventional for protobuf (apparently, can update if proved wrong)
// alternative, consider float32, since we only ever measure 0.05x changes

func GetFloatParams(params *structpb.Struct, key string, def float64) float64 {
	if params == nil{
		return def
	}

	val, ok := params.Fields[key]
	if !ok {
		return def
	}

	if num:= val.GetNumberValue(); num != 0 || val.Kind == nil{
		return num
	}

	return def
}

func GetStringParam(params *structpb.Struct, key string, def string) string {
	if params == nil{
		return def
	}
	val, ok := params.Fields[key]
	if !ok {
		return def
	}

	return val.GetStringValue()
}

func GetBoolParam(params *structpb.Struct, key string, def bool) bool {
	if params == nil{
		return def
	}
	val, ok := params.Fields[key]

	if !ok {
		return def
	}
	return val.GetBoolValue()
}