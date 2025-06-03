package scala

import (
	"log"
	"time"
	"sort"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/language"
	"github.com/stackb/scala-gazelle/pkg/scalaconfig"
	"github.com/bazelbuild/bazel-gazelle/rule"
)

const debugGenerate = false

func filterStrSlice(elts []string, f func(string) bool) []string {
	var out []string
	for _, elt := range elts {
		if !f(elt) {
			continue
		}
		out = append(out, elt)
	}
	return out
}

// GenerateRules implements part of the language.Language interface
func (sl *scalaLang) GenerateRules(args language.GenerateArgs) language.GenerateResult {

	// if args.File == nil {
	// 	return language.GenerateResult{}
	// }

	t1 := time.Now()

	if sl.wantProgress && sl.cache.PackageCount > 0 {
		writeGenerateProgress(sl.progress, len(sl.packages), int(sl.cache.PackageCount))
	}

	sc := scalaconfig.Get(args.Config)
	pkg := newScalaPackage(args, sc, sl.ruleProviderRegistry, sl.parser, sl)
	sl.packages[args.Rel] = pkg
	sl.remainingPackages++

	// generating bazel rules - TODO move elsewhere 
	collectedFiles := filterStrSlice(args.RegularFiles, func(f string) bool { return filepath.Ext(f) == ".scala" })
	if len(collectedFiles) == 0 {
		return language.GenerateResult{}
	}
	sort.Strings(collectedFiles)
	scalaTestRules := make([]*rule.Rule, 0)
	scalaLibraryFiles := make([]string, 0)
	for _, file := range collectedFiles {
		if sc.IsScalaTestFile(filepath.Base(file)) {
			addedScalaTestRule := sl.generateScalaTest(args.File, args.Rel, file, pkg)
			addedScalaTestRule.SetPrivateAttr(ruleProviderKey, pkg.addedNewRule(addedScalaTestRule))
			scalaTestRules = append(scalaTestRules, addedScalaTestRule)
		} else {
			scalaLibraryFiles = append(scalaLibraryFiles, file)
		}
	}
	scalaLibRule := sl.generateScalaLibrary(args.File, args.Rel, filepath.Base(args.Rel), scalaLibraryFiles, pkg)
	scalaLibRule.SetPrivateAttr(ruleProviderKey, pkg.addedNewRule(scalaLibRule))
	// generating bazel rules end

	rules := pkg.Rules()
	rules = append(rules, scalaLibRule)
	for _, testRule := range scalaTestRules {
		rules = append(rules, testRule)
	}

	for _, r := range rules {
		from := label.Label{Pkg: args.Rel, Name: r.Name()}
		sl.PutKnownRule(from, r)
	}

	rules = append(rules, generatePackageMarkerRule(len(sl.packages), pkg))

	imports := make([]interface{}, len(rules))
	for i, r := range rules {
		imports[i] = r.PrivateAttr(config.GazelleImportsKey)
	}

	if debugGenerate {
		t2 := time.Since(t1).Round(1 * time.Millisecond)
		if len(rules) > 1 {
			log.Printf("Visited %q (%d rules, %v)", args.Rel, len(rules)-1, t2)
		}
	}

	return language.GenerateResult{
		Gen:     rules,
		Imports: imports,
	}
}

func (sl *scalaLang) generateScalaLibrary(file *rule.File, pathToPackageRelativeToBazelWorkspace string, name string, srcsRelativeToBazelWorkspace []string, pkg *scalaPackage) *rule.Rule {
	const ruleKind = "scala_library"
	r := rule.NewRule(ruleKind, name)

	srcs := make([]string, 0, len(srcsRelativeToBazelWorkspace))
	for _, src := range srcsRelativeToBazelWorkspace {
		srcs = append(srcs, strings.TrimPrefix(filepath.ToSlash(src), filepath.ToSlash(pathToPackageRelativeToBazelWorkspace+"/")))
	}
	sort.Strings(srcs)

	r.SetAttr("srcs", srcs)
	defaultVisibility := "//:__subpackages__"
	if pkg.cfg.DefaultVisibility() != "" {
		defaultVisibility = pkg.cfg.DefaultVisibility()
	}
	r.SetAttr("visibility", []string{defaultVisibility})

	return r
}

func (sl *scalaLang) generateScalaTest(file *rule.File, pathToPackageRelativeToBazelWorkspace string, testSrcRelativeToBazelWorkspace string, pkg *scalaPackage) *rule.Rule {
	const ruleKind = "scala_test"
	name := strings.TrimSuffix(testSrcRelativeToBazelWorkspace, ".scala")
	r := rule.NewRule(ruleKind, name)

	srcs := []string{strings.TrimPrefix(filepath.ToSlash(testSrcRelativeToBazelWorkspace), filepath.ToSlash(pathToPackageRelativeToBazelWorkspace+"/"))}

	r.SetAttr("srcs", srcs)

	return r
}
