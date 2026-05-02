package module_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kristofer/picoceci/pkg/bytecode"
	"github.com/kristofer/picoceci/pkg/eval"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/module"
	"github.com/kristofer/picoceci/pkg/parser"
)

func getTestdataPath() string {
	// Find the testdata directory relative to the project root
	wd, _ := os.Getwd()
	// Go up from pkg/module to project root
	return filepath.Join(wd, "..", "..", "testdata")
}

func TestIntegrationLoadModuleFromFile(t *testing.T) {
	testdata := getTestdataPath()
	modulesPath := filepath.Join(testdata, "modules")

	resolver := module.NewResolver(os.ReadFile)
	resolver.SetSDCardRoot(modulesPath + "/")
	loader := module.NewLoader(resolver)

	// Load Counter module
	mod, err := loader.Load("Counter")
	if err != nil {
		t.Fatalf("failed to load Counter: %v", err)
	}

	// Verify the module has a Counter global
	if _, ok := mod.Globals["Counter"]; !ok {
		t.Error("expected Counter global in loaded module")
	}
}

func TestIntegrationCircularImportDetection(t *testing.T) {
	testdata := getTestdataPath()
	modulesPath := filepath.Join(testdata, "modules")

	resolver := module.NewResolver(os.ReadFile)
	resolver.SetSDCardRoot(modulesPath + "/")
	loader := module.NewLoader(resolver)

	// Try to load Circular1 which imports Circular2 which imports Circular1
	_, err := loader.Load("Circular1")
	if err == nil {
		t.Fatal("expected circular import error")
	}
	if !strings.Contains(err.Error(), "circular import") {
		t.Errorf("expected 'circular import' in error, got: %v", err)
	}
}

func TestIntegrationBuiltinModules(t *testing.T) {
	resolver := module.NewResolver(os.ReadFile)
	module.RegisterBuiltins(resolver)
	loader := module.NewLoader(resolver)

	// Load built-in core module
	_, err := loader.Load("core")
	if err != nil {
		t.Fatalf("failed to load core builtin: %v", err)
	}

	// Load built-in io module
	_, err = loader.Load("io")
	if err != nil {
		t.Fatalf("failed to load io builtin: %v", err)
	}

	// Load built-in collections module
	_, err = loader.Load("collections")
	if err != nil {
		t.Fatalf("failed to load collections builtin: %v", err)
	}
}

func TestIntegrationBytecodeCompilerWithImport(t *testing.T) {
	testdata := getTestdataPath()
	modulesPath := filepath.Join(testdata, "modules")

	resolver := module.NewResolver(os.ReadFile)
	resolver.SetSDCardRoot(modulesPath + "/")
	loader := module.NewLoader(resolver)

	// Compile code that imports Counter
	src := `import 'Counter'. | c | c := Counter new.`
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	compiler := bytecode.NewCompilerWithLoader(loader)
	_, compileErr := compiler.Compile(prog.Statements)
	if compileErr != nil {
		t.Fatalf("compile error: %v", compileErr)
	}

	// Verify Counter was imported
	globals := compiler.GetGlobals()
	if _, ok := globals["Counter"]; !ok {
		t.Error("expected Counter in compiler globals after import")
	}
}

func TestIntegrationEvalInterpreterWithImport(t *testing.T) {
	testdata := getTestdataPath()
	modulesPath := filepath.Join(testdata, "modules")

	resolver := module.NewResolver(os.ReadFile)
	resolver.SetSDCardRoot(modulesPath + "/")
	loader := module.NewLoader(resolver)

	// Create interpreter with loader
	interp := eval.NewWithLoader(loader)

	// Eval code that imports Counter
	src := `import 'Counter'.`
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	_, evalErr := interp.Eval(prog.Statements)
	if evalErr != nil {
		t.Fatalf("eval error: %v", evalErr)
	}
}

func TestIntegrationModuleCaching(t *testing.T) {
	testdata := getTestdataPath()
	modulesPath := filepath.Join(testdata, "modules")

	loadCount := 0
	countingReader := func(path string) ([]byte, error) {
		loadCount++
		return os.ReadFile(path)
	}

	resolver := module.NewResolver(countingReader)
	resolver.SetSDCardRoot(modulesPath + "/")
	loader := module.NewLoader(resolver)

	// Load Counter twice
	_, err := loader.Load("Counter")
	if err != nil {
		t.Fatalf("first load error: %v", err)
	}
	firstCount := loadCount

	_, err = loader.Load("Counter")
	if err != nil {
		t.Fatalf("second load error: %v", err)
	}

	// Should not have read the file again
	if loadCount != firstCount {
		t.Errorf("expected module to be cached, but file was read again")
	}
}

func TestIntegrationSDCardPath(t *testing.T) {
	testdata := getTestdataPath()
	sdcardPath := filepath.Join(testdata, "sdcard", "picoceci", "libs") + "/"

	resolver := module.NewResolver(os.ReadFile)
	resolver.SetSDCardRoot(sdcardPath)
	loader := module.NewLoader(resolver)

	// Load module from simulated SD card
	mod, err := loader.Load("SDCardModule")
	if err != nil {
		t.Fatalf("failed to load SDCardModule: %v", err)
	}

	if mod.Path != sdcardPath+"SDCardModule.pc" {
		t.Errorf("unexpected path: %s", mod.Path)
	}
}
