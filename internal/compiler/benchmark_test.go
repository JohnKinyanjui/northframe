package compiler

import "testing"

func BenchmarkCompileIndependentComponent(b *testing.B) {
	source := []byte(`---
interface Props {
Title string = "Untitled"
Items []string
}
---
<section><h1>{Props.Title}</h1>{for item := range Props.Items}<p>{item}</p>{/for}<slot /></section>`)
	known := map[string]componentView{}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := compileComponent("routes", "Panel", source, known); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuildUtilityStyles(b *testing.B) {
	sources := [][]byte{[]byte(`<main class="grid min-h-screen gap-6 bg-stone-50 p-6 text-stone-950 md:grid-cols-2 xl:grid-cols-4"><button class="rounded-full bg-lime-300 px-5 py-3 hover:bg-lime-400">Save</button></main>`)}
	b.ReportAllocs()
	for b.Loop() {
		_ = BuildStyles(sources, nil)
	}
}
