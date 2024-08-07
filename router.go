package goblaze

type Router interface {
	//TODO our handler Type
	// Use(args ...any) Router

	// Get(path string, handler Handler, middleware ...Handler) Router
	// Head(path string, handler Handler, middleware ...Handler) Router
	// Post(path string, handler Handler, middleware ...Handler) Router
	// Put(path string, handler Handler, middleware ...Handler) Router
	// Delete(path string, handler Handler, middleware ...Handler) Router
	// Connect(path string, handler Handler, middleware ...Handler) Router
	// Options(path string, handler Handler, middleware ...Handler) Router
	// Trace(path string, handler Handler, middleware ...Handler) Router
	// Patch(path string, handler Handler, middleware ...Handler) Router

	// Add(methods []string, path string, handler Handler, middleware ...Handler) Router
	// All(path string, handler Handler, middleware ...Handler) Router

	// Group(prefix string, handlers ...Handler) Router

	// Route(path string) Register

	// Name(name string) Router
}
