package proto

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"math/big"
)

type DHState struct {
	G, Q, P *big.Int
}

var (
	ErrParseDHState = errors.New("failed to parse second user's diffie-hellman state")
	ErrParseHalfKey = errors.New("failed to parse second user's halfkey")
	ErrWeakHalfkey  = errors.New("weak halfkey from second user")

	ErrShortQ = errWeakDHState("q constant size lower than 256 bits")
	ErrShortP = errWeakDHState("p constant size lower than 2048 bits")
	ErrWrongG = errWeakDHState("g^q (modp) doesn't equal 1")
)

func InitDHState() (*DHState, error) {
	q, err := rand.Prime(rand.Reader, 256)
	if err != nil {
		return nil, err
	}

	N, p, err := genNP(q)
	if err != nil {
		return nil, err
	}

	g, err := genG(N, p, q)
	if err != nil {
		return nil, err
	}

	return &DHState{
		G: g,
		Q: q,
		P: p,
	}, nil
}

func (dh *DHState) ActiveDHExchange(conn *Conn) ([]byte, error) {
	// Send dhstate
	buf, _ := json.Marshal(dh)
	_, err := conn.WritePackage(buf)
	if err != nil {
		return nil, errExchangeIO(err)
	}

	// Generate X. halfkeyX = g^X (modp)
	X, err := rand.Int(rand.Reader, dh.Q)
	if err != nil {
		return nil, errExchangeIO(err)
	}

	halfkeyX := big.NewInt(0)
	X.Mod(X, dh.Q)
	halfkeyX.Exp(dh.G, X, dh.P)

	// Send halfkeyX
	buf, _ = halfkeyX.MarshalText()
	_, err = conn.WritePackage(buf)
	if err != nil {
		return nil, errExchangeIO(err)
	}

	// Recieve halfkeyY
	buf, err = conn.ReadPackage()
	if err != nil {
		return nil, errExchangeIO(err)
	}
	halfkeyY := big.NewInt(0)
	err = halfkeyY.UnmarshalText(buf)
	if err != nil {
		return nil, ErrParseHalfKey
	}

	// Check halfkeyY
	if !dh.CheckHalfkey(halfkeyY) {
		return nil, ErrWeakHalfkey
	}

	// Calculate key. key = g^XY (modp)
	halfkeyY.Exp(halfkeyY, X, dh.P)
	return halfkeyY.Bytes(), nil
}

func (dh *DHState) PassiveDHExchange(conn *Conn) ([]byte, error) {
	// Recieve DHState
	buf, err := conn.ReadPackage()
	if err != nil {
		return nil, errExchangeIO(err)
	}

	err = json.Unmarshal(buf, dh)
	if err != nil {
		return nil, ErrParseDHState
	}

	// Check DHState
	err = dh.CheckDHState()
	if err != nil {
		return nil, err
	}

	// Recieve halfkeyX
	buf, err = conn.ReadPackage()
	if err != nil {
		return nil, errExchangeIO(err)
	}
	halfkeyX := big.NewInt(0)
	err = halfkeyX.UnmarshalText(buf)
	if err != nil {
		return nil, ErrParseHalfKey
	}

	// Check halfkeyX
	if !dh.CheckHalfkey(halfkeyX) {
		return nil, ErrWeakHalfkey
	}

	// Generate Y. halfkeyY = g^Y (modp)
	Y, err := rand.Int(rand.Reader, dh.Q)
	if err != nil {
		return nil, errExchangeIO(err)
	}
	Y.Mod(Y, dh.Q)
	halfkeyY := big.NewInt(0)
	halfkeyY.Exp(dh.G, Y, dh.P)

	// Send halfkeyY
	buf, _ = halfkeyY.MarshalText()
	_, err = conn.WritePackage(buf)
	if err != nil {
		return nil, errExchangeIO(err)
	}

	// Generate key. key = g^XY (modp)
	halfkeyX.Exp(halfkeyX, Y, dh.P)
	return halfkeyX.Bytes(), nil
}

func (dh *DHState) CheckDHState() error {
	if dh.Q.BitLen() < 256 {
		return ErrShortQ
	}

	if dh.P.BitLen() < 2048 {
		return ErrShortP
	}

	test := big.NewInt(0)
	test.Exp(dh.G, dh.Q, dh.P)

	if test.Cmp(big.NewInt(1)) != 0 {
		return ErrWrongG
	}

	return nil
}

func (dh *DHState) CheckHalfkey(halfkey *big.Int) bool {
	test := big.NewInt(0)
	test.Exp(halfkey, dh.Q, dh.P)
	return (test.Cmp(big.NewInt(1)) == 0)
}

func genNP(q *big.Int) (*big.Int, *big.Int, error) {
	var N, p *big.Int
	var err error

	p = big.NewInt(0)
	for i := 0; i < 10000; i++ {
		N, err = genIntBits(1792)
		if err != nil {
			continue
		}

		p.Mul(N, q)
		p.Add(p, big.NewInt(1))
		if p.ProbablyPrime(64) {
			return N, p, nil
		}
	}

	return nil, nil, errors.New("failed to generate N")
}

func genG(N, p, q *big.Int) (*big.Int, error) {
	var a, g *big.Int
	var err error

	g = big.NewInt(0)
	for i := 0; i < 10000; i++ {
		a, err = rand.Int(rand.Reader, p)
		if err != nil {
			continue
		}

		g.Exp(a, N, p)
		a.Exp(g, q, p)

		if a.Cmp(big.NewInt(1)) == 0 {
			return g, nil
		}
	}

	return nil, errors.New("failed to generate g")
}

func genIntBits(bits uint) (*big.Int, error) {
	max := big.NewInt(1)
	max.Lsh(max, bits)

	for i := 0; i < 100; i++ {
		result, err := rand.Int(rand.Reader, max)
		if err != nil {
			continue
		}

		if uint(result.BitLen()) == bits {
			return result, nil
		}
	}

	return nil, errors.New("failed to generate int")
}

func errExchangeIO(err error) error {
	return errors.New("I/O error while key exchange: " + err.Error())
}

func errWeakDHState(msg string) error {
	return errors.New("weak diffie-hellman state from second user: " + msg)
}
