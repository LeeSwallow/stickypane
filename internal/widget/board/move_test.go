package board

import (
	"strings"
	"testing"

	"github.com/LeeSwallow/stickypane/internal/doc"
)

const moveSrc = "## To do\n- payments\n  - stripe first\n- login API\n## Doing\n## Done\n- schema\n"

func TestMoveFindsTheCardAndTheColumnByName(t *testing.T) {
	d, err := (Move{Card: "login", To: "doing"}).Apply(doc.Parse([]byte(moveSrc)))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(d.Bytes()); got != "## To do\n- payments\n  - stripe first\n## Doing\n- login API\n## Done\n- schema\n" {
		t.Errorf("Move = %q", got)
	}
	d, err = (Move{Card: "payments", To: "Done"}).Apply(doc.Parse([]byte(moveSrc)))
	if err != nil || !strings.HasSuffix(string(d.Bytes()), "## Done\n- schema\n- payments\n  - stripe first\n") {
		t.Errorf("a card moves with its details: %v\n%s", err, d.Bytes())
	}
	same, err := (Move{Card: "schema", To: "Done"}).Apply(doc.Parse([]byte(moveSrc)))
	if err != nil || string(same.Bytes()) != moveSrc {
		t.Errorf("moving a card to where it is changes nothing: %v", err)
	}
	if _, err := (Move{Card: "schema", To: "Shipped"}).Apply(doc.Parse([]byte(moveSrc))); err == nil || !strings.Contains(err.Error(), "Doing") {
		t.Errorf("a column that is not there should list the columns: %v", err)
	}
	if _, err := (Move{Card: "nothing", To: "Done"}).Apply(doc.Parse([]byte(moveSrc))); err == nil || !strings.Contains(err.Error(), "payments") {
		t.Errorf("a card that is not there should list the cards: %v", err)
	}
}

func TestAddPutsACardInAColumn(t *testing.T) {
	d, err := (Add{Card: "refunds", To: "doing"}).Apply(doc.Parse([]byte(moveSrc)))
	if err != nil || !strings.Contains(string(d.Bytes()), "## Doing\n- refunds\n## Done\n") {
		t.Errorf("Add = %v\n%s", err, d.Bytes())
	}
	d, err = (Add{Card: "refunds"}).Apply(doc.Parse([]byte(moveSrc)))
	if err != nil || !strings.Contains(string(d.Bytes()), "- login API\n- refunds\n## Doing\n") {
		t.Errorf("without a column the card goes to the first one: %v\n%s", err, d.Bytes())
	}
	d, err = (Add{Card: "first", To: "Backlog"}).Apply(doc.Parse([]byte("---\ntype: board\n---\n")))
	if err != nil || string(d.Bytes()) != "---\ntype: board\n---\n## Backlog\n- first\n" {
		t.Errorf("a column that is not there yet is made: %v\n%q", err, d.Bytes())
	}
	d, err = (Add{Card: "first"}).Apply(doc.Parse([]byte("")))
	if err != nil || string(d.Bytes()) != "## To do\n- first\n" {
		t.Errorf("an empty board gets a first column: %v\n%q", err, d.Bytes())
	}
}
