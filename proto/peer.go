package proto

type Peer struct {
	Login  string
	Addr   string
	DifHel *DHState     `json:"-"`
	Crypto *CryptoState `json:"-"`
	Conn   *Conn        `json:"-"`
}

func (p *Peer) WritePackage(buf []byte) (int, error) {
	enc, err := p.Crypto.AuthAndEncrypt(buf)
	if err != nil {
		return 0, err
	}

	return p.Conn.WritePackage(enc)
}

func (p *Peer) ReadPackage() ([]byte, error) {
	enc, err := p.Conn.ReadPackage()
	if err != nil {
		return nil, err
	}

	return p.Crypto.DecryptAndAuth(enc)
}

func (p *Peer) Close() {
	p.Conn.Close()
}
