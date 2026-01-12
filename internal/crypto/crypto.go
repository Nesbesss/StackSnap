package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)


const (
	VersionCTR byte = 0x01
	VersionGCM byte = 0x02
)


var MagicHeader = []byte("SSNAP")


const GCMChunkSize = 64 * 1024


var (
	ErrInvalidHeader    = errors.New("invalid encryption header")
	ErrUnsupportedVersion  = errors.New("unsupported encryption version")
	ErrAuthenticationFailed = errors.New("authentication failed: backup may be corrupted or tampered")
)


type EncryptedHeader struct {
	Magic  [5]byte
	Version byte
	Nonce  [12]byte
}


type gcmWriter struct {
	aead  cipher.AEAD
	nonce  []byte
	counter uint64
	w    io.Writer
	buf   []byte
}

func (g *gcmWriter) Write(p []byte) (int, error) {
	g.buf = append(g.buf, p...)
	written := 0

	for len(g.buf) >= GCMChunkSize {
		if err := g.flushChunk(g.buf[:GCMChunkSize]); err != nil {
			return written, err
		}
		written += GCMChunkSize
		g.buf = g.buf[GCMChunkSize:]
	}

	return len(p), nil
}

func (g *gcmWriter) flushChunk(chunk []byte) error {

	chunkNonce := make([]byte, 12)
	copy(chunkNonce, g.nonce[:4])
	binary.BigEndian.PutUint64(chunkNonce[4:], g.counter)
	g.counter++


	ciphertext := g.aead.Seal(nil, chunkNonce, chunk, nil)


	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(ciphertext)))
	if _, err := g.w.Write(lenBuf); err != nil {
		return err
	}
	if _, err := g.w.Write(ciphertext); err != nil {
		return err
	}

	return nil
}

func (g *gcmWriter) Close() error {

	if len(g.buf) > 0 {
		if err := g.flushChunk(g.buf); err != nil {
			return err
		}
		g.buf = nil
	}


	endMarker := make([]byte, 4)
	_, err := g.w.Write(endMarker)
	return err
}



func NewEncryptWriterGCM(key []byte, w io.Writer) (io.WriteCloser, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}


	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}


	header := EncryptedHeader{
		Version: VersionGCM,
	}
	copy(header.Magic[:], MagicHeader)
	copy(header.Nonce[:], nonce)

	if _, err := w.Write(header.Magic[:]); err != nil {
		return nil, fmt.Errorf("failed to write magic: %w", err)
	}
	if _, err := w.Write([]byte{header.Version}); err != nil {
		return nil, fmt.Errorf("failed to write version: %w", err)
	}
	if _, err := w.Write(header.Nonce[:]); err != nil {
		return nil, fmt.Errorf("failed to write nonce: %w", err)
	}

	return &gcmWriter{
		aead: aead,
		nonce: nonce,
		w:   w,
		buf:  make([]byte, 0, GCMChunkSize),
	}, nil
}


type gcmReader struct {
	aead  cipher.AEAD
	nonce  []byte
	counter uint64
	r    io.Reader
	buf   []byte
	eof   bool
}

func (g *gcmReader) Read(p []byte) (int, error) {

	if len(g.buf) > 0 {
		n := copy(p, g.buf)
		g.buf = g.buf[n:]
		return n, nil
	}

	if g.eof {
		return 0, io.EOF
	}


	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(g.r, lenBuf); err != nil {
		return 0, err
	}

	chunkLen := binary.BigEndian.Uint32(lenBuf)
	if chunkLen == 0 {
		g.eof = true
		return 0, io.EOF
	}


	ciphertext := make([]byte, chunkLen)
	if _, err := io.ReadFull(g.r, ciphertext); err != nil {
		return 0, err
	}


	chunkNonce := make([]byte, 12)
	copy(chunkNonce, g.nonce[:4])
	binary.BigEndian.PutUint64(chunkNonce[4:], g.counter)
	g.counter++


	plaintext, err := g.aead.Open(nil, chunkNonce, ciphertext, nil)
	if err != nil {
		return 0, ErrAuthenticationFailed
	}

	n := copy(p, plaintext)
	if n < len(plaintext) {
		g.buf = plaintext[n:]
	}

	return n, nil
}


func NewDecryptReaderGCM(key []byte, r io.Reader) (io.Reader, error) {

	magic := make([]byte, 5)
	if _, err := io.ReadFull(r, magic); err != nil {
		return nil, fmt.Errorf("failed to read magic: %w", err)
	}
	if !bytes.Equal(magic, MagicHeader) {
		return nil, ErrInvalidHeader
	}

	versionBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, versionBuf); err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}

	if versionBuf[0] != VersionGCM {
		return nil, fmt.Errorf("%w: got version %d", ErrUnsupportedVersion, versionBuf[0])
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(r, nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &gcmReader{
		aead: aead,
		nonce: nonce,
		r:   r,
	}, nil
}



func NewEncryptWriter(key []byte, w io.Writer) (io.WriteCloser, error) {
	return NewEncryptWriterGCM(key, w)
}



func NewDecryptReader(key []byte, r io.Reader) (io.Reader, error) {

	peek := make([]byte, 6)
	n, err := io.ReadFull(r, peek)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}


	if n >= 5 && bytes.Equal(peek[:5], MagicHeader) {
		version := peek[5]
		switch version {
		case VersionGCM:

			nonce := make([]byte, 12)
			if _, err := io.ReadFull(r, nonce); err != nil {
				return nil, fmt.Errorf("failed to read nonce: %w", err)
			}

			block, err := aes.NewCipher(key)
			if err != nil {
				return nil, fmt.Errorf("failed to create cipher: %w", err)
			}

			aead, err := cipher.NewGCM(block)
			if err != nil {
				return nil, fmt.Errorf("failed to create GCM: %w", err)
			}

			return &gcmReader{
				aead: aead,
				nonce: nonce,
				r:   r,
			}, nil

		case VersionCTR:

			return nil, fmt.Errorf("CTR version in new header format not supported")

		default:
			return nil, fmt.Errorf("%w: version %d", ErrUnsupportedVersion, version)
		}
	}



	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	iv := make([]byte, block.BlockSize())
	copy(iv, peek[:n])
	remaining := block.BlockSize() - n
	if remaining > 0 {
		if _, err := io.ReadFull(r, iv[n:]); err != nil {
			return nil, fmt.Errorf("failed to read legacy IV: %w", err)
		}
	}

	stream := cipher.NewCTR(block, iv)
	return &cipher.StreamReader{S: stream, R: r}, nil
}
