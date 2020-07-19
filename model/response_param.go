package model

type ResponseParam struct {
	Protocol string
	Status   string
	Headers  map[string][]string
	BodyBuff []byte
}
