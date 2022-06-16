package mp4

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"io"
)

// DecryptBytesCTR - decrypt or encrypt sample using CTR mode
func DecryptBytesCTR(data []byte, key []byte, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(block, iv)

	inBuf := bytes.NewBuffer(data)
	outBuf := bytes.Buffer{}

	writer := cipher.StreamWriter{S: stream, W: &outBuf}
	_, err = io.Copy(writer, inBuf)
	if err != nil {
		return nil, err
	}
	return outBuf.Bytes(), nil
}

// DecryptSampleCenc - decrypt cenc-schema encrypted sample provided key, iv, and subSamplePatterns
func DecryptSampleCenc(sample []byte, key []byte, iv []byte, subSamplePatterns []SubSamplePattern) ([]byte, error) {
	decSample := make([]byte, 0, len(sample))
	if len(subSamplePatterns) != 0 {
		var pos uint32 = 0
		for j := 0; j < len(subSamplePatterns); j++ {
			ss := subSamplePatterns[j]
			nrClear := uint32(ss.BytesOfClearData)
			if nrClear > 0 {
				decSample = append(decSample, sample[pos:pos+nrClear]...)
				pos += nrClear
			}
			nrEnc := ss.BytesOfProtectedData
			if nrEnc > 0 {
				cryptOut, err := DecryptBytesCTR(sample[pos:pos+nrEnc], key, iv)
				if err != nil {
					return nil, err
				}
				decSample = append(decSample, cryptOut...)
				pos += nrEnc
			}
		}
	} else {
		cryptOut, err := DecryptBytesCTR(sample, key, iv)
		if err != nil {
			return nil, err
		}
		decSample = append(decSample, cryptOut...)
	}
	return decSample, nil
}

// DecryptSampleCenc - decrypt cenc-schema encrypted sample in place provided key, iv, and subSamplePatterns
func DecryptSampleCbcs(sample []byte, key []byte, iv []byte, subSamplePatterns []SubSamplePattern, tenc *TencBox) error {
	nrInCryptBlock := int(tenc.DefaultCryptByteBlock) * 16
	nrInSkipBlock := int(tenc.DefaultSkipByteBlock) * 16
	var pos uint32 = 0
	if len(subSamplePatterns) != 0 {
		for j := 0; j < len(subSamplePatterns); j++ {
			ss := subSamplePatterns[j]
			nrClear := uint32(ss.BytesOfClearData)
			pos += nrClear
			if ss.BytesOfProtectedData > 0 {
				err := cbcsDecrypt(sample[pos:pos+ss.BytesOfProtectedData], key,
					iv, nrInCryptBlock, nrInSkipBlock)
				if err != nil {
					return err
				}
			}
			pos += ss.BytesOfProtectedData
		}
	} else { // Full cbcs - this should not happen since the first part should be in clear
		err := cbcsDecrypt(sample, key, iv, nrInCryptBlock, nrInSkipBlock)
		if err != nil {
			return err
		}
	}
	return nil
}

// cbcDecrypt - in place striped or full CBC decryption. Full if nrInSkipBlock == 0
func cbcsDecrypt(data []byte, key []byte, iv []byte, nrInCryptBlock, nrInSkipBlock int) error {
	pos := 0
	size := len(data) // This is the bytes that we should stripe decrypt
	aesCbcCrypto, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	dec := cipher.NewCBCDecrypter(aesCbcCrypto, iv)
	if nrInSkipBlock == 0 {
		nrToDecrypt := size & ^0xf // Drops 4 last bits -> multiple of 16
		dec.CryptBlocks(data[:nrToDecrypt], data[:nrToDecrypt])
		return nil
	}
	for {
		if size-pos < nrInCryptBlock { // Leave the rest
			break
		}
		dec.CryptBlocks(data[pos:pos+nrInCryptBlock], data[pos:pos+nrInCryptBlock])
		pos += nrInCryptBlock
		if size-pos < nrInSkipBlock {
			break
		}
		pos += nrInSkipBlock
	}
	return nil
}
