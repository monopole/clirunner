package cli

// Constants needed by both the fake shell and its db.
const (
	// TestCliPath expected to be on the PATH when tests run.
	TestCliPath       = "testcli"
	FlagDisablePrompt = "disable-prompt"
	FlagExitOnErr     = "exit-on-error"
	FlagNumRowsInDb   = "num-rows-in-db"
	FlagRowToErrorOn  = "row-to-error-on"
)

// In earlier versions, the testcli behaved more like a program called "mql",
// because the problem being solved was "how can we run mql".  The process
// runner was tightly bound to parsing, meaning its tests had to cover both the
// runner and parsing, thus testcli had to behave like mql.
//
// This is no longer the case; the runner is tested independently
// of any parsers, and the parsers have their own tests.  The mentions
// of mql below are vestigial.
//goland:noinspection SpellCheckingInspection
const (
	delimiter           = "_|_"
	nodeScanQueryWord   = "query"
	nodeLookupQueryWord = "bus"
	versionOf3DX        = "3DEXPERIENCE R2018x HotFix 8"
	requestedErrFmt     = "error! touching row %d triggers this error"
)
