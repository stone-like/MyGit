package database

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mygit/src/crypt"
	c "mygit/src/database/content"
	con "mygit/src/database/content"
	"mygit/util"
	"os"
	"path/filepath"
	"strings"
)

type Database struct {
	Path string
	Objs map[string]c.Object
}

func (d *Database) CreateContent(o c.Object) string {
	bytes := []byte(o.ToString())
	content := fmt.Sprintf("%s %d\x00%s", o.Type(), len(bytes), bytes)
	return content
}
func (d *Database) SetObjId(o c.Object, content string) {
	o.SetObjId(crypt.HexDigestBySha1(content))
}

func GetStoreHeaderContent(o c.Object) string {
	bytes := []byte(o.ToString())
	content := fmt.Sprintf("%s %d\x00%s", o.Type(), len(bytes), bytes)

	return content
}

func (d *Database) Store(o c.Object) {
	content := GetStoreHeaderContent(o)
	o.SetObjId(crypt.HexDigestBySha1(content))
	d.WriteObject(o.GetObjId(), content)
}

func (d *Database) ObjPath(objId string) string {

	return filepath.Join(d.Path, objId[0:2], objId[2:])
}

func (d *Database) WriteObject(objId, content string) error {
	// objPath := filepath.Join(d.Path, objId[0:2], objId[2:])
	// dirName := filepath.Dir(filepath.Clean(objPath))

	objPath := d.ObjPath(objId)

	if _, err := os.Stat(objPath); err == nil {
		//内容によってディレクトリ、ファイル名が決定するのですでに存在していたらもう作る必要ない
		return nil
	}

	var in bytes.Buffer
	err := CompressWithDeflate(&in, content)

	if err != nil {
		return err
	}

	util.WriteFile(objPath, &in)

	return nil
}

func CompressWithDeflate(in *bytes.Buffer, content string) error {
	b := []byte(content)
	w, err := zlib.NewWriterLevel(in, 1)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	w.Close()
	if err != nil {
		return err
	}

	return nil

}

func Inflate(w io.Writer, r io.Reader) error {
	r, err := zlib.NewReader(r)

	if err != nil {
		return err
	}
	io.Copy(w, r)

	return nil
}

var ErrorUnexpectedObjType = errors.New("unexpectedObjType")

func (d *Database) GenerateTree(objId string) (string, error) {
	var retContent string

	o, err := d.ReadObject(objId)
	if err != nil {
		return "", err
	}

	//まず一番初めにCommitをParseしてTreeを入手しそのEntriesを使うので、、ここにはEntryであるTreeかBlobしか来ない想定
	t, ok := o.(*c.Tree)
	if !ok {
		return "", ErrorUnexpectedObjType
	}

	for _, v := range t.Entries {
		e, ok := v.(*c.Entry)
		if !ok {
			return "", ErrorUnexpectedObjType
		}

		if e.IsTree() {
			ret, err := d.GenerateTree(e.ObjId)
			if err != nil {
				return "", err
			}
			retContent += ret
		} else {
			content := fmt.Sprintf("%o %s %s\n", e.Mode, e.ObjId, e.Path)
			retContent += content
		}
	}

	return retContent, nil

}

func (d *Database) ShowTree(objId string, w io.Writer) error {

	str, err := d.GenerateTree(objId)

	if err != nil {
		return err
	}

	w.Write([]byte(str))

	return nil

}

func (d *Database) ReadObject(objId string) (c.ParsedObj, error) {

	r, err := d.GetContent(objId)
	if err != nil {
		return nil, err
	}

	bufR := bufio.NewReader(r)

	hAndR, err := d.ScanObjectHeader(bufR)
	if err != nil {
		return nil, err
	}
	o, err := ParseObjectContent(hAndR.ObjType, hAndR.Reader)
	if err != nil {
		return nil, err
	}

	o.SetObjId(objId)

	return o, nil

}

func ParseObjectContent(objType string, r io.Reader) (c.ParsedObj, error) {
	obj, err := c.Parse(objType, r)
	if err != nil {
		return nil, err
	}

	return obj, err
}

func (d *Database) GetContent(objId string) (io.Reader, error) {
	objPath := d.ObjPath(objId)
	if _, err := os.Stat(objPath); err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(objPath)
	if err != nil {
		return nil, err
	}

	in := bytes.NewBuffer(data)

	out := new(bytes.Buffer)

	err = Inflate(out, in)

	if err != nil {
		return nil, err
	}

	return out, nil

}

type ObjHeaderAndReader struct {
	Size    string
	ObjType string
	Reader  io.Reader
}

func (d *Database) ScanObjectHeader(r io.Reader) (*ObjHeaderAndReader, error) {
	//関数型みたいに逐一Readerを返すといいかも？
	b := bufio.NewReader(r)

	//blob 4testみたいになっているので、まずはtypeとsizeを読み込む
	objType, err := b.ReadString(' ')
	if err != nil {
		return nil, err
	}

	sizePlusNullTerm, err := b.ReadString('\x00')
	if err != nil {
		return nil, err
	}

	size := sizePlusNullTerm[:len(sizePlusNullTerm)-1]

	return &ObjHeaderAndReader{
		Reader:  b,
		Size:    size,
		ObjType: strings.TrimSpace(objType),
	}, nil

}

func (d *Database) ShortObjId(objId string) string {
	return objId[0:6]
}

func (d *Database) ObjDirname(name string) string {
	return filepath.Join(d.Path, name[0:2])
}

//mに指定したPathのTree以下のすべてのBlobを追加
func (d *Database) BuildList(path string, m map[string]*con.Entry, e *con.Entry) (map[string]*con.Entry, error) {

	if e == nil {
		return nil, nil
	}

	//Blobの時
	if !e.IsTree() {
		m[path] = e
		return m, nil
	}

	//Treeの時
	o, err := d.ReadObject(e.GetObjId())
	if err != nil {
		return nil, err
	}
	t, ok := o.(*con.Tree)
	if !ok {
		return nil, ErrorObjeToEntryConvError
	}

	for _, o := range t.Entries {
		e, ok := o.(*con.Entry)

		if !ok {
			return nil, ErrorObjeToEntryConvError
		}

		_, err := d.BuildList(e.Path, m, e)
		if err != nil {
			return nil, err
		}
	}

	return m, err

}

func (d *Database) LoadTreeList(objId string) (map[string]*con.Entry, error) {
	return d.RunLoadTreeList(objId, "")
}

func (d *Database) LoadTreeListWithPath(objId, path string) (map[string]*con.Entry, error) {
	return d.RunLoadTreeList(objId, path)
}

func (d *Database) RunLoadTreeList(objId, path string) (map[string]*con.Entry, error) {

	if objId == "" {
		return nil, nil
	}

	e, err := d.RunLoadTreeEntry(objId, path)
	if err != nil {
		return nil, err
	}

	m := make(map[string]*con.Entry)

	buildedMap, err := d.BuildList(path, m, e)
	if err != nil {
		return nil, err
	}

	return buildedMap, err

}

//pathなしならCommitからTree,pathありならTreeからそのpathのBlobを返す(両方ともEntryの形として)

func (d *Database) LoadTreeEntry(objId string) (*con.Entry, error) {
	return d.RunLoadTreeEntry(objId, "")
}
func (d *Database) LoadTreeEntryWithPath(objId, path string) (*con.Entry, error) {
	return d.RunLoadTreeEntry(objId, path)
}
func (d *Database) RunLoadTreeEntry(objId, path string) (*con.Entry, error) {
	o, err := d.ReadObject(objId)
	if err != nil {
		return nil, err
	}

	c, ok := o.(*con.CommitFromMem)
	if !ok {
		return nil, ErrorObjeToEntryConvError
	}

	rootTreeEntry := &con.Entry{
		ObjId: c.Tree,
		Mode:  con.ModeToInt(con.DIRECTORY_MODE),
	}

	if path == "" {
		return rootTreeEntry, nil
	}

	var currentEntry con.Object //基本的にTree,最後の1ループでcurrentEntryにBlobがセットされる
	currentEntry = rootTreeEntry
	//DescendをrelativePathで使う想定
	for _, p := range util.Descend(path) {
		if currentEntry == nil {
			break
		}
		o, err := d.ReadObject(currentEntry.GetObjId())
		if err != nil {
			return nil, err
		}
		// 最後の1ループでcurrentEntryにBlobがセットされる、それまでは全部TreeなのでここでTreeConversionをしてよい
		t, ok := o.(*con.Tree)
		if !ok {
			return nil, ErrorObjeToEntryConvError
		}

		newEntry := t.Entries[p]

		currentEntry = newEntry

	}

	//blobだろうが、treeだろうがEntryの形であることには変わりない
	e, entryOk := currentEntry.(*con.Entry)

	if !entryOk {
		return nil, ErrorObjeToEntryConvError
	}

	return e, nil

}
