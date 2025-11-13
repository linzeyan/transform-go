package convert

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"
)

type uuid [16]byte

var (
	uuidMutex    sync.Mutex
	uuidNodeID   [6]byte
	uuidClockSeq uint16
	uuidLastTime uint64
)

const uuidEpoch = 122192928000000000

var uuidNamespaces = map[string]uuid{
	"DNS":  {0x6b, 0xa7, 0xb8, 0x10, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8},
	"URL":  {0x6b, 0xa7, 0xb8, 0x11, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8},
	"OID":  {0x6b, 0xa7, 0xb8, 0x12, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8},
	"X500": {0x6b, 0xa7, 0xb8, 0x14, 0x9d, 0xad, 0x11, 0xd1, 0x80, 0xb4, 0x00, 0xc0, 0x4f, 0xd4, 0x30, 0xc8},
}

func init() {
	if _, err := rand.Read(uuidNodeID[:]); err != nil {
		panic(err)
	}
	uuidNodeID[0] |= 0x01
	var b [2]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	uuidClockSeq = binary.BigEndian.Uint16(b[:]) & 0x3fff
}

// GenerateUUIDs returns UUID v1~v8, GUID, and ULID.
func GenerateUUIDs() (map[string]string, error) {
	out := make(map[string]string, 10)
	var err error
	if out["v1"], err = uuidV1(); err != nil {
		return nil, err
	}
	if out["v2"], err = uuidV2(); err != nil {
		return nil, err
	}
	if out["v3"], err = uuidNameBased(3); err != nil {
		return nil, err
	}
	if out["v4"], err = uuidV4(); err != nil {
		return nil, err
	}
	if out["v5"], err = uuidNameBased(5); err != nil {
		return nil, err
	}
	if out["v6"], err = uuidV6(); err != nil {
		return nil, err
	}
	if out["v7"], err = uuidV7(); err != nil {
		return nil, err
	}
	if out["v8"], err = uuidV8(); err != nil {
		return nil, err
	}
	if out["guid"], err = generateGUID(); err != nil {
		return nil, err
	}
	if out["ulid"], err = generateULID(); err != nil {
		return nil, err
	}
	return out, nil
}

func uuidV1() (string, error) {
	ts, seq := nextUUIDState()
	u := buildUUIDv1(ts, seq)
	return u.String(), nil
}

func uuidV2() (string, error) {
	var u uuid
	if _, err := rand.Read(u[:]); err != nil {
		return "", err
	}
	u[6] = (u[6] & 0x0f) | 0x20
	setVariant(&u)
	return u.String(), nil
}

func uuidV4() (string, error) {
	var u uuid
	if _, err := rand.Read(u[:]); err != nil {
		return "", err
	}
	u[6] = (u[6] & 0x0f) | 0x40
	setVariant(&u)
	return u.String(), nil
}

func uuidV6() (string, error) {
	ts, seq := nextUUIDState()
	v1 := buildUUIDv1(ts, seq)
	var reordered uuid
	reordered[0] = v1[6]
	reordered[1] = v1[7]
	reordered[2] = v1[4]
	reordered[3] = v1[5]
	reordered[4] = v1[0]
	reordered[5] = v1[1]
	reordered[6] = v1[2]
	reordered[7] = v1[3]
	copy(reordered[8:], v1[8:])
	reordered[6] = (reordered[6] & 0x0f) | 0x60
	setVariant(&reordered)
	return reordered.String(), nil
}

func uuidV7() (string, error) {
	ms := uint64(time.Now().UnixMilli())
	var u uuid
	u[0] = byte(ms >> 40)
	u[1] = byte(ms >> 32)
	u[2] = byte(ms >> 24)
	u[3] = byte(ms >> 16)
	u[4] = byte(ms >> 8)
	u[5] = byte(ms)
	if _, err := rand.Read(u[6:]); err != nil {
		return "", err
	}
	u[6] = (u[6] & 0x0f) | 0x70
	setVariant(&u)
	return u.String(), nil
}

func uuidV8() (string, error) {
	var u uuid
	if _, err := rand.Read(u[:]); err != nil {
		return "", err
	}
	u[6] = (u[6] & 0x0f) | 0x80
	setVariant(&u)
	return u.String(), nil
}

func uuidNameBased(version int) (string, error) {
	ns := uuidNamespaces["DNS"]
	name := make([]byte, 32)
	if _, err := rand.Read(name); err != nil {
		return "", err
	}
	var sum []byte
	if version == 3 {
		h := md5.New()
		h.Write(ns[:])
		h.Write(name)
		sum = h.Sum(nil)
	} else {
		h := sha1.New()
		h.Write(ns[:])
		h.Write(name)
		sum = h.Sum(nil)
	}
	var u uuid
	copy(u[:], sum[:16])
	u[6] = (u[6] & 0x0f) | byte(version<<4)
	setVariant(&u)
	return u.String(), nil
}

func generateGUID() (string, error) {
	id, err := uuidV4()
	if err != nil {
		return "", err
	}
	return strings.ToUpper(id), nil
}

func generateULID() (string, error) {
	ms := uint64(time.Now().UnixMilli())
	var data [16]byte
	for i := 0; i < 6; i++ {
		data[i] = byte(ms >> (40 - 8*i))
	}
	if _, err := rand.Read(data[6:]); err != nil {
		return "", err
	}
	return encodeULID(data[:]), nil
}

func encodeULID(data []byte) string {
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	value := uint64(0)
	bits := 0
	out := make([]byte, 0, 26)
	for _, b := range data {
		value = (value << 8) | uint64(b)
		bits += 8
		for bits >= 5 {
			bits -= 5
			out = append(out, alphabet[(value>>bits)&0x1f])
		}
	}
	if bits > 0 {
		out = append(out, alphabet[(value<<(5-bits))&0x1f])
	}
	for len(out) < 26 {
		out = append(out, alphabet[0])
	}
	if len(out) > 26 {
		out = out[:26]
	}
	return string(out)
}

func nextUUIDState() (uint64, uint16) {
	uuidMutex.Lock()
	defer uuidMutex.Unlock()
	ts := uint64(time.Now().UnixNano()/100) + uuidEpoch
	if ts <= uuidLastTime {
		uuidClockSeq = (uuidClockSeq + 1) & 0x3fff
		ts = uuidLastTime + 1
	}
	uuidLastTime = ts
	return ts, uuidClockSeq
}

func buildUUIDv1(timestamp uint64, seq uint16) uuid {
	var u uuid
	binary.BigEndian.PutUint32(u[0:], uint32(timestamp))
	binary.BigEndian.PutUint16(u[4:], uint16(timestamp>>32))
	binary.BigEndian.PutUint16(u[6:], uint16(timestamp>>48))
	u[6] = (u[6] & 0x0f) | 0x10
	binary.BigEndian.PutUint16(u[8:], seq)
	copy(u[10:], uuidNodeID[:])
	setVariant(&u)
	return u
}

func setVariant(u *uuid) {
	u[8] = (u[8] & 0x3f) | 0x80
}

func (u uuid) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(u[0:4]),
		binary.BigEndian.Uint16(u[4:6]),
		binary.BigEndian.Uint16(u[6:8]),
		binary.BigEndian.Uint16(u[8:10]),
		u[10:16],
	)
}
