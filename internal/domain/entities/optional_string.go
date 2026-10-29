package entities

import "encoding/json"

// OptionalString distinguishes an omitted JSON field from an explicit null.
type OptionalString struct {
	Set bool
	Value *string
}

func (o OptionalString) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.Set = true
	return json.Unmarshal(data, &o.Value)
}
