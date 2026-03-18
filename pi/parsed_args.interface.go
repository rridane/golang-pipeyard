package pi

type ParsedArgsInterface interface {
	/* -------- String -------- */
	StringOk(name string) (string, bool)
	String(name string) string
	RequireString(name string) (string, error)

	/* -------- Int -------- */
	IntOk(name string) (int, bool)
	Int(name string) int
	RequireInt(name string) (int, error)

	/* -------- Bool -------- */
	BoolOk(name string) (bool, bool)
	Bool(name string) bool
	RequireBool(name string) (bool, error)

	/* -------- Keys -------- */
	Keys() (stringsKeys, boolKeys, intKeys []string)

	/* -------- Setters -------- */
	SetString(name, value string)
	SetBool(name string, value bool)
	SetInt(name string, value int)
}
