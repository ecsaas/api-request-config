package arcg

type Error struct {
	Field string      `json:"field"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type ErrorTypeList []Error        //arcg
type Store map[string]interface{} //arcg

type ServerAlert struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}
