# flowhash

The flowhash Go package provides Community ID flow hashing.

See https://github.com/corelight/community-id-spec

## Usage

```golang
import "github.com/adriansr/flowhash"

func ExampleCommunityIDHash() {
	flow := flowhash.Flow{
		SourceIP:        net.ParseIP("10.1.2.3"),
		DestinationIP:   net.ParseIP("8.8.8.8"),
		SourcePort:      63521,
		DestinationPort: 53,
		Protocol:        17,
	}
	fmt.Println(flowhash.CommunityID.Hash(flow))
	// Output: 1:R7iR6vkxw+jaz3wjDfWMWooBdfc=
}
```
