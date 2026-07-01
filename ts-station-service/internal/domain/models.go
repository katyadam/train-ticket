package domain

import "encoding/json"

type Response struct {
	Status int    `json:"status"`
	Msg    string `json:"msg"`
	Data   any    `json:"data"`
}

type Station struct {
	ID       *string `json:"id"`
	Name     *string `json:"name"`
	StayTime int     `json:"stayTime"`
}

// UnmarshalJSON preserves the Java entity constructor's empty-name default while
// still distinguishing an explicit JSON null, which Jackson rejects in setName.
func (s *Station) UnmarshalJSON(data []byte) error {
	empty := ""
	type station Station
	value := station{Name: &empty}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*s = Station(value)
	return nil
}
