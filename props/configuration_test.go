package props

import(
	"testing"
	"model"
)

type Properties struct {
	c model.Config
	e error
}

var props Properties

func TestReadConfiguration(t *testing.T) {

	rd := "../"
	cfPath := rd + "sq.properties"
	cfg, err := GetConfiguration(cfPath)

	props = Properties{c : cfg, e : err}
	if err != nil {
		t.Error(err.Error())
	}
}

func TestMandatoryProperties(t *testing.T) {

	if props.e == nil {
		if props.c.Proto == "" {
			t.Error("PROTO missing in sq.properties\n")
		}
		if len(props.c.Endpoints) == 0 {
			t.Error("ENDPOINTS missing in sq.properties\n")
		}
		if props.c.ConcurrencyPeak == 0 {
			t.Error("CONCURRENCY_PEAK missing in sq.properties\n")
		}
	}
}
