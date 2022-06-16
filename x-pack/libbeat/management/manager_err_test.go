package management

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/pkg/errors"
	pkgErrors "github.com/pkg/errors"
)

// []error len: 1, cap: 1, [
//         *github.com/pkg/errors.withStack {
//                 error: error(*github.com/pkg/errors.withMessage) *{
//                         cause: error(*github.com/elastic/beats/v7/libbeat/common.ErrInputNotFinished) *{
//                                 State: "{Id: native::5223-36, Finished: false, Fileinfo: &{c.log 25151507 420 {235918666 63790789002 0x76342c0} {36 5223 1 33188 1000 1000 0 0 25151507 4096 49128 {1655191984 408543519} {1655192202 235918666} {1655192202 235918666} [0 0 0]}}, Source: /tmp/flog/c.log, Offset: 25225045, Timestamp: 2022-06-14 07:42:33.726063285 +0000 UTC m=+352.385800275, TTL: -1ns, Type: log, Meta: map[], FileStateOS: 5223-36}",},
//                         msg: "Error creating runner from config",},
//                 stack: *github.com/pkg/errors.stack len: 6, cap: 32, [36271317,76706549,76703411,76699962,36002601,25244097],},
// 			]

func TestErrorStruff(t *testing.T) {
	root := common.ErrInputNotFinished{State: "bar state"}
	msg := pkgErrors.WithMessage(&root, "foo message")
	stack := pkgErrors.WithStack(msg)
	t.Log(errors.Is(stack, &(common.ErrInputNotFinished{})))
}
