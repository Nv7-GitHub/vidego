package vidego

import (
	"image"
	"io"

	"github.com/3d0c/gmf"
)

func NewDecoder(path string) (*Decoder, error) {
	// Open file
	ctx, err := gmf.NewInputCtx(path)
	if err != nil {
		return nil, err
	}

	// Get video stream
	stream, err := ctx.GetBestStream(gmf.AVMEDIA_TYPE_VIDEO)
	if err != nil {
		return nil, err
	}

	// Get codec
	codec, err := gmf.FindEncoder(gmf.AV_CODEC_ID_H264)
	if err != nil {
		return nil, err
	}

	// Set up codec context
	cc := gmf.NewCodecCtx(codec)
	cc.SetTimeBase(gmf.AVR{Num: 1, Den: 1})
	cc.SetPixFmt(gmf.AV_PIX_FMT_RGBA).SetWidth(stream.CodecCtx().Width()).SetHeight(stream.CodecCtx().Height())
	if codec.IsExperimental() {
		cc.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
	}

	// Open Codec
	err = cc.Open(nil)
	if err != nil {
		return nil, err
	}

	// Get IST
	ist, err := ctx.GetStream(stream.Index())
	if err != nil {
		return nil, err
	}

	icc := stream.CodecCtx()
	swsctx, err := gmf.NewSwsCtx(icc.Width(), icc.Height(), icc.PixFmt(), cc.Width(), cc.Height(), cc.PixFmt(), gmf.SWS_BICUBIC)
	if err != nil {
		return nil, err
	}

	return &Decoder{
		ctx:    ctx,
		stream: stream,
		cc:     cc,
		ist:    ist,
		swsctx: swsctx,
		drain:  -1,
	}, nil
}

type Decoder struct {
	ctx    *gmf.FmtCtx
	stream *gmf.Stream
	cc     *gmf.CodecCtx
	ist    *gmf.Stream
	swsctx *gmf.SwsCtx

	pkt    *gmf.Packet
	frames []*gmf.Frame
	drain  int
}

func (d *Decoder) Size() (int, int) {
	return d.cc.Width(), d.cc.Height()
}

func (d *Decoder) FrameCount() int {
	return d.stream.NbFrames()
}

func (d *Decoder) GetNextFrame() (bool, []image.Image, error) {
	var err error
	// Get next packet
	d.pkt, err = d.ctx.GetNextPacket()
	if err != nil && err != io.EOF {
		if d.pkt != nil {
			d.pkt.Free()
		}
		return false, nil, err
	} else if err != nil && d.pkt == nil {
		d.drain = 0
	}

	// Decode frame
	d.frames, err = d.ist.CodecCtx().Decode(d.pkt)
	if err != nil {
		return false, nil, err
	}

	// Handle error
	if d.pkt != nil && d.pkt.StreamIndex() != d.stream.Index() {
		return true, nil, nil
	}

	// Load frames
	d.frames, err = gmf.DefaultRescaler(d.swsctx, d.frames)
	if err != nil {
		return false, nil, err
	}

	// Encode packets
	pkts, err := d.cc.Encode(d.frames, d.drain)
	if err != nil {
		return false, nil, err
	}
	if len(pkts) == 0 {
		return true, nil, nil
	}

	// Convert to Go images
	out := make([]image.Image, len(pkts))
	w, h := d.Size()
	for i, p := range pkts {
		img := new(image.RGBA)
		img.Pix = p.Data()
		img.Stride = 4 * w
		img.Rect = image.Rect(0, 0, w, h)
		out[i] = img
		p.Free()
	}

	// Cleanup
	for i := range d.frames {
		d.frames[i].Free()
	}

	return true, out, nil
}

func (d *Decoder) Free() {
	for i := 0; i < d.ctx.StreamsCnt(); i++ {
		st, _ := d.ctx.GetStream(i)
		st.CodecCtx().Free()
		st.Free()
	}
	d.stream.Free()
	d.ctx.Free()
	d.cc.Free()
	gmf.Release(d.cc)
	d.ist.Free()
	d.swsctx.Free()
}
