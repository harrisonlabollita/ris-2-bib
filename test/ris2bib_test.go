package ris2bib


import (  
	"github.com/harrisonlabollita/ris-2-bib"
    "testing"
)

func TestCreateBib(t *testing.T){

    var filename string = "test.ris"

    data, err := os.ReadFile(filename);
    if err != nil { log.Fatal(err) }

    bib_ref := bib_entry{ " ",
                          " ",
                          {" ", " ", " "},

    }


}
