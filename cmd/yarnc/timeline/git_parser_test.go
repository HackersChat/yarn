package timeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	_ "go.yarn.social/lextwt"
	"go.yarn.social/types"
)

type testPrinter struct {
	Lines []string
}

func (t *testPrinter) Print(format string, a ...interface{}) {
	t.Lines = append(t.Lines, fmt.Sprintf(format, a...))
}

func TestParser(t *testing.T) {
	mainTwter := types.Twter{
		Nick: "nick",
	}

	mainNode1 := types.MakeTwt(mainTwter, time.Now(), "1")
	mainNode2 := types.MakeTwt(mainTwter, time.Now(), fmt.Sprintf("(#%s) 1.2", mainNode1.Hash()))
	mainNode3 := types.MakeTwt(mainTwter, time.Now(), fmt.Sprintf("(#%s) 1.2.2", mainNode2.Hash()))

	tests := []struct {
		name  string
		twts  types.Twts
		wants []string
	}{
		{
			name: "1 - When only root nodes",
			twts: []types.Twt{
				types.MakeTwt(mainTwter, time.Now(), "Hello"),
				types.MakeTwt(mainTwter, time.Now(), "World"),
				types.MakeTwt(mainTwter, time.Now(), "!"),
			},
			wants: []string{
				`\* .*- Hello`,
				`\* .*- World`,
				`\* .*- !`,
			},
		},
		{
			name: "2- * and simple branch",
			twts: []types.Twt{
				mainNode1,
				types.MakeTwt(mainTwter, time.Now(), fmt.Sprintf("(#%s) 1.1", mainNode1.Hash())),
				types.MakeTwt(mainTwter, time.Now(), fmt.Sprintf("(#%s) 1.2", mainNode1.Hash())),
			},
			wants: []string{
				`\* .*- 1`,
				`| \\`,
				`| \* .* 1.1`,
				`| \* .* 1.2`,
			},
		},
		{
			name: "3- Simple branch with * in the middle",
			twts: []types.Twt{
				mainNode1,
				types.MakeTwt(mainTwter, time.Now(), fmt.Sprintf("(#%s) 1.1", mainNode1.Hash())),
				types.MakeTwt(mainTwter, time.Now(), "Hello"),
				types.MakeTwt(mainTwter, time.Now(), fmt.Sprintf("(#%s) 1.2", mainNode1.Hash())),
			},
			wants: []string{
				`\* .*- 1`,
				`| \\`,
				`| \* .* 1.1`,
				`| \* .* 1.2`,
				`\* .*- Hello`,
			},
		},
		{
			name: "4- Triple Level",
			twts: []types.Twt{
				mainNode1,
				mainNode2,
				types.MakeTwt(mainTwter, time.Now(), fmt.Sprintf("(#%s) 1.3", mainNode1.Hash())),
				types.MakeTwt(mainTwter, time.Now(), fmt.Sprintf("(#%s) 1.2.1", mainNode2.Hash())),
				mainNode3,
				types.MakeTwt(mainTwter, time.Now(), fmt.Sprintf("(#%s) 1.2.2.1", mainNode3.Hash())),
			},
			wants: []string{
				`\* .*- 1`,
				`| \\`,
				`| \* .* 1.2`,
				`| | \\`,
				`| | \* .* 1.2.1`,
				`| | \* .* 1.2.2`,
				`| | | \\`,
				`| | | \* .* 1.2.2.1`,
				`| \* .* 1.3`,
			},
		},
		{
			name: "5- Child without Parent",
			twts: []types.Twt{
				types.MakeTwt(mainTwter, time.Now(), "Hello"),
				types.MakeTwt(mainTwter, time.Now(), "(#abcdefg) World"),
			},
			wants: []string{
				`\* .*- Hello`,
				`\* .* World`,
			},
		},
	}

	for _, test := range tests {
		printer := testPrinter{}
		gp := gitParser{&printer}
		gp.Parse(context.Background(), test.twts, mainTwter)
		assert.Equal(t, len(test.wants), len(printer.Lines))
		if len(test.wants) == len(printer.Lines) {
			for i := range printer.Lines {
				assert.Regexpf(t, test.wants[i], printer.Lines[i], test.name)
			}
		}
	}

}
