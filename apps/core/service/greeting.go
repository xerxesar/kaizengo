package service

// Greeting is the core business use-case shared by SSR and GraphQL.
type Greeting struct{}

func NewGreeting() *Greeting { return &Greeting{} }

func (g *Greeting) Hello(name string) string {
	if name == "" {
		name = "world"
	}
	return "Hello, " + name + "!"
}
