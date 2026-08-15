package docs

// Response is the API documentation envelope listing every module.
type Response struct {
	Modules []Module `json:"modules"`
}

// Module describes one API module and its routes.
type Module struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Routes      []Route `json:"routes"`
}

// Route describes one HTTP endpoint.
type Route struct {
	Method       string  `json:"method"`
	Path         string  `json:"path"`
	Summary      string  `json:"summary"`
	Description  string  `json:"description"`
	Auth         string  `json:"auth"`
	PathParams   []Field `json:"path_params,omitempty"`
	RequestBody  string  `json:"request_body,omitempty"`
	ResponseBody string  `json:"response_body,omitempty"`
	Errors       []Error `json:"errors,omitempty"`
}

// Field describes one path parameter of a route.
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Error describes one error a route may return.
type Error struct {
	Status      int    `json:"status"`
	Code        string `json:"code"`
	Description string `json:"description"`
}
