package runtime

type RuntimeContext struct {
    Vars map[string]string
}

func NewRuntimeContext() *RuntimeContext {
    return &RuntimeContext{
        Vars: make(map[string]string),
    }
}

func (c *RuntimeContext) Set(name, value string) {
    c.Vars[name] = value
}

func (c *RuntimeContext) Get(name string) string {
    return c.Vars[name]
}
