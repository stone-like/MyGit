package src

//diff3はいったんdiffアルゴリズムを自分で実装してから
type Diff3 struct {
	origin    string
	a         string
	b         string
	chunks    []string
	matchA    []string
	matchB    []string
	lineOrign int
	lineA     int
	lineB     int
}

func (d *Diff3) Merge() {

}

func SetUp() {

}
