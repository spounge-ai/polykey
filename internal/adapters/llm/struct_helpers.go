package llm

import (
	"google.golang.org/protobuf/types/known/structpb"
)

// ParamExtractor handles parameter extraction from protobuf Struct
type ParamExtractor struct {
	fields map[string]*structpb.Value
}

func NewParamExtractor(params *structpb.Struct) *ParamExtractor {
	if params == nil {
		return &ParamExtractor{fields: make(map[string]*structpb.Value)}
	}
	return &ParamExtractor{fields: params.Fields}
}

// Parameter extractors with default fallback
func (p *ParamExtractor) Float64(key string, def float64) float64 {
	if val := p.fields[key]; val != nil {
		return val.GetNumberValue()
	}
	return def
}

func (p *ParamExtractor) String(key string, def string) string {
	if val := p.fields[key]; val != nil {
		return val.GetStringValue()
	}
	return def
}

func (p *ParamExtractor) Bool(key string, def bool) bool {
	if val := p.fields[key]; val != nil {
		return val.GetBoolValue()
	}
	return def
}

func (p *ParamExtractor) Int(key string, def int) int {
	if val := p.fields[key]; val != nil {
		return int(val.GetNumberValue())
	}
	return def
}

func (p *ParamExtractor) Int32(key string, def int32) int32 {
	if val := p.fields[key]; val != nil {
		return int32(val.GetNumberValue())
	}
	return def
}

func (p *ParamExtractor) Int64(key string, def int64) int64 {
	if val := p.fields[key]; val != nil {
		return int64(val.GetNumberValue())
	}
	return def
}

// Complex parameter extractors
func (p *ParamExtractor) StringSlice(key string, def []string) []string {
	val := p.fields[key]
	if val == nil {
		return def
	}
	
	listVal := val.GetListValue()
	if listVal == nil {
		return def
	}
	
	if len(listVal.Values) == 0 {
		return []string{}
	}
	
	result := make([]string, len(listVal.Values))
	for i, v := range listVal.Values {
		result[i] = v.GetStringValue()
	}
	return result
}

func (p *ParamExtractor) Map(key string, def map[string]interface{}) map[string]interface{} {
	val := p.fields[key]
	if val == nil {
		return def
	}
	
	structVal := val.GetStructValue()
	if structVal == nil {
		return def
	}
	
	if len(structVal.Fields) == 0 {
		return make(map[string]interface{})
	}
	
	result := make(map[string]interface{}, len(structVal.Fields))
	for k, v := range structVal.Fields {
		result[k] = convertValue(v)
	}
	return result
}

// convertValue converts a protobuf Value to a Go interface{}
func convertValue(val *structpb.Value) interface{} {
	switch kind := val.Kind.(type) {
	case *structpb.Value_NumberValue:
		return kind.NumberValue
	case *structpb.Value_StringValue:
		return kind.StringValue
	case *structpb.Value_BoolValue:
		return kind.BoolValue
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_ListValue:
		values := kind.ListValue.Values
		if len(values) == 0 {
			return []interface{}{}
		}
		result := make([]interface{}, len(values))
		for i, v := range values {
			result[i] = convertValue(v)
		}
		return result
	case *structpb.Value_StructValue:
		fields := kind.StructValue.Fields
		if len(fields) == 0 {
			return make(map[string]interface{})
		}
		result := make(map[string]interface{}, len(fields))
		for k, v := range fields {
			result[k] = convertValue(v)
		}
		return result
	default:
		return nil
	}
}

// Convenience functions for backward compatibility
func GetFloatParam(params *structpb.Struct, key string, def float64) float64 {
	if params != nil {
		if val := params.Fields[key]; val != nil {
			return val.GetNumberValue()
		}
	}
	return def
}

func GetStringParam(params *structpb.Struct, key string, def string) string {
	if params != nil {
		if val := params.Fields[key]; val != nil {
			return val.GetStringValue()
		}
	}
	return def
}

func GetBoolParam(params *structpb.Struct, key string, def bool) bool {
	if params != nil {
		if val := params.Fields[key]; val != nil {
			return val.GetBoolValue()
		}
	}
	return def
}

func GetIntParam(params *structpb.Struct, key string, def int) int {
	if params != nil {
		if val := params.Fields[key]; val != nil {
			return int(val.GetNumberValue())
		}
	}
	return def
}

func GetStringSliceParam(params *structpb.Struct, key string, def []string) []string {
	return NewParamExtractor(params).StringSlice(key, def)
}