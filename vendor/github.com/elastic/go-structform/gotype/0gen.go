// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
//go:generate mktmpl -f -o unfold_user_primitive.generated.go unfold_user_primitive.yml
//go:generate mktmpl -f -o unfold_user_processing.generated.go unfold_user_processing.yml

// go:generate mktmpl -f -o unfold_sel_generated.go unfold_sel.yml
