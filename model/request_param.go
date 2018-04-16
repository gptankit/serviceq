package model

type RequestParam struct {
	Protocol   string
	Method     string
	RequestURI string
	Headers    map[string][]string
	BodyBuff   []byte
}
