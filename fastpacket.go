package nmea

import (
	"sync"
	"time"
)

type Assembler interface {
	Assemble(frame RawFrame, to *RawMessage) bool
}

type fastPacketSequence struct {
	header CanBusHeader

	lastReceivedFrameTime time.Time
	// sequence is message counter to distinguish to which message frame belongs. 0-7. Frames from same source may arrive
	// out of order and without sequence counter it is hard to know if in which message this frame belongs.
	sequence uint8
	// length of data in all frames. Length is found as second byte in first frame
	length             uint8
	completeFramesMask uint32

	// Fast-Packet data is maximum of 32 frames. First frame 6 bytes and max 31 frame of 7 bytes. Last frame can be 1-7 bytes.
	receivedFramesMask  uint32 // each frame is single bit
	receivedFramesCount uint8
	data                [FastRawPacketMaxSize]byte
}

func (m *fastPacketSequence) Append(frame RawFrame) bool {
	if frame.Length < 2 {
		return false
	}
	sequence := frame.Data[0] >> 5 // last 3 bits (sequence counter range is 0-7)

	frameNr := frame.Data[0] & 0b0001_1111 // first 5 bits
	frameMask := uint32(1 << (frameNr))
	if m.receivedFramesMask&frameMask != 0 { // we have already seen that frame
		// maybe should be error? can we receive same frame more than once?
		return m.completeFramesMask == m.receivedFramesMask
	}
	if m.receivedFramesMask == 0 {
		m.header = frame.Header
		m.sequence = sequence
	}
	m.receivedFramesMask |= frameMask
	m.receivedFramesCount++
	m.lastReceivedFrameTime = frame.Time

	if frameNr == 0 { // first frame initializes lengths ,so we know when sequence is complete
		// very first frame 0th, has 2 bytes for metadata (3 bits sequence counter, 5bits frame counter, 8bits length)
		// and 6 bytes actual data
		m.length = frame.Data[1]

		frameCount := uint8(1)
		if m.length > 6 { // fast packet data is multiple frames long
			frameCount += (m.length - 6 + 7) / 7
		}
		m.completeFramesMask = ^(0xFFFFFFFF << frameCount)

		copy(m.data[:6], frame.Data[2:])
	} else { // subsequent frames, have 7 bytes of data, first byte is for sequence counter and frame counter
		start := 6 + int(frameNr-1)*7
		end := start + len(frame.Data) - 1
		copy(m.data[start:end], frame.Data[1:])
	}

	return m.completeFramesMask == m.receivedFramesMask
}

func (m *fastPacketSequence) Reset() {
	m.lastReceivedFrameTime = time.Time{}

	m.header.PGN = 0
	m.header.Priority = 0
	m.header.Source = 0
	m.header.Destination = 0

	m.sequence = 0
	m.length = 0
	m.completeFramesMask = 0
	m.receivedFramesMask = 0
	m.receivedFramesCount = 0
	// we do not reset data here. data will be overridden
}

func (m *fastPacketSequence) To(to *RawMessage) {
	to.Time = m.lastReceivedFrameTime
	to.Header = m.header

	if cap(to.Data) < int(m.length) {
		to.Data = make([]byte, m.length)
	}
	copy(to.Data[:], m.data[0:m.length])
}

func (m *fastPacketSequence) As() RawMessage {
	data := make([]byte, m.length)
	copy(data[:], m.data[0:m.length])

	return RawMessage{
		Time:   m.lastReceivedFrameTime,
		Header: m.header,
		Data:   data,
	}
}

type FastPacketAssembler struct {
	// pgns is list of PGNs that are transferred as Fast-Packet RawFrame and should be assembled to RawMessage
	pgns       []uint32
	inTransfer []*fastPacketSequence

	now  func() time.Time
	pool *sync.Pool
	lock sync.Mutex
}

func NewFastPacketAssembler(fpPGNs []uint32) *FastPacketAssembler {
	pool := new(sync.Pool)
	pool.New = func() any {
		return &fastPacketSequence{}
	}

	return &FastPacketAssembler{
		pgns:       append([]uint32{}, fpPGNs...),
		inTransfer: make([]*fastPacketSequence, 0, 10),

		now:  time.Now,
		pool: pool,
	}
}

func (a *FastPacketAssembler) Assemble(frame RawFrame, to *RawMessage) bool {
	a.lock.Lock()
	defer a.lock.Unlock()

	isFastPacket := false
	if couldBeFastPacket(frame.Header.PGN) {
		for _, pgn := range a.pgns {
			if pgn == frame.Header.PGN {
				isFastPacket = true
				break
			}
		}
	}
	if !isFastPacket {
		if cap(to.Data) < int(frame.Length) {
			to.Data = make([]byte, frame.Length)
		}
		copy(to.Data[:], frame.Data[0:frame.Length])
		to.Time = frame.Time
		to.Header = frame.Header
		return true
	}

	// fast packet sequence is uniquely identified by: source+pgn+sequence+lastReceivedFrameTime

	threshold := a.now().Add(-750 * time.Millisecond)
	sequence := frame.Data[0] >> 5 // last 3 bits (sequence counter range is 0-7)

	var fp *fastPacketSequence
	idx := 0
	for i, tmpFp := range a.inTransfer {
		if tmpFp.header.Source != frame.Header.Source ||
			tmpFp.header.PGN != frame.Header.PGN ||
			tmpFp.sequence != sequence {
			continue
		}
		fp = a.inTransfer[i]
		idx = i
		if fp.lastReceivedFrameTime.Before(threshold) { // sequence is too old to be this frame sequence
			fp.Reset()
		}
	}
	if fp == nil {
		fp = a.pool.Get().(*fastPacketSequence)
		fp.Reset()
		a.inTransfer = append(a.inTransfer, fp)
		idx = len(a.inTransfer) - 1
	}
	isComplete := fp.Append(frame)
	if isComplete { // message is now complete
		fp.To(to) // copy data over to rawMessage

		// remove item from in transfer list and put it back to pool
		a.inTransfer[idx] = a.inTransfer[len(a.inTransfer)-1]
		a.inTransfer = a.inTransfer[:len(a.inTransfer)-1]
		a.pool.Put(fp)
	} else {
		a.inTransfer[idx] = fp
	}
	return isComplete
}
