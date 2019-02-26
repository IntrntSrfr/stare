package owo

type OWOResult struct {
	Success     bool   `json:"success"`
	ErrorCode   int    `json:"errorcode"`
	Description string `json:"description"`
	Files       []struct {
		Hash string `json:"hash"`
		Name string `json:"name"`
		URL  string `json:"url"`
		Size int    `json:"size"`
	} `json:"files"`
}
