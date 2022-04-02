package log

import (
	"io"
	"os"

	"github.com/tysonmote/gommap"
)

var (
	OffWidth   uint64 = 4
	PosWidth   uint64 = 8
	EntryWidth        = OffWidth + PosWidth
)

type Index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
}

func NewIndex(f *os.File, c Config) (*Index, error) {
	idx := &Index{
		file: f,
	}

	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	idx.size = uint64(fi.Size())
	if err := os.Truncate(
		f.Name(), int64(c.Segment.MaxIndexBytes),
	); err != nil {
		return nil, err
	}
	if idx.mmap, err = gommap.Map(
		idx.file.Fd(),
		gommap.PROT_READ|gommap.PROT_WRITE,
		gommap.MAP_SHARED,
	); err != nil {
		return nil, err
	}
	return idx, nil
}

func (i *Index) Close() error {
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return err
	}

	if err := i.file.Sync(); err != nil {
		return err
	}
	if err := i.file.Truncate(int64(i.size)); err != nil {
		return err
	}
	return i.file.Close()
}

func (i *Index) Read(in int64) (out uint32, pos uint64, err error) {

	if in == -1 {
		out = uint32((i.size / EntryWidth) - 1)
	} else {
		out = uint32(in)
	}
	pos = uint64(out) * EntryWidth
	if i.size == 0 || i.size < pos+EntryWidth {
		return 0, 0, io.EOF
	} else {
		out = Enc.Uint32(i.mmap[pos : pos+OffWidth])
		pos = Enc.Uint64(i.mmap[pos+OffWidth : pos+EntryWidth])
		return out, pos, nil
	}

}

func (i *Index) Write(off uint32, pos uint64) error {
	if uint64(len(i.mmap)) < i.size+EntryWidth {
		return io.EOF
	}
	Enc.PutUint32(i.mmap[i.size:i.size+OffWidth], off)
	Enc.PutUint64(i.mmap[i.size+OffWidth:i.size+EntryWidth], pos)
	i.size += uint64(EntryWidth)
	return nil
}

func (i *Index) Name() string {
	return i.file.Name()
}
