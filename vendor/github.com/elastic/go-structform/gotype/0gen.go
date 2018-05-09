package gotype

//go:generate mktmpl -f -o fold_map_inline.generated.go fold_map_inline.yml
//go:generate mktmpl -f -o fold_refl_sel.generated.go fold_refl_sel.yml
//go:generate mktmpl -f -o stacks.generated.go stacks.yml
//go:generate mktmpl -f -o unfold_primitive.generated.go unfold_primitive.yml
//go:generate mktmpl -f -o unfold_lookup_go.generated.go unfold_lookup_go.yml
//go:generate mktmpl -f -o unfold_err.generated.go unfold_err.yml
//go:generate mktmpl -f -o unfold_arr.generated.go unfold_arr.yml
//go:generate mktmpl -f -o unfold_map.generated.go unfold_map.yml
//go:generate mktmpl -f -o unfold_refl.generated.go unfold_refl.yml
//go:generate mktmpl -f -o unfold_ignore.generated.go unfold_ignore.yml

// go:generate mktmpl -f -o unfold_sel_generated.go unfold_sel.yml
