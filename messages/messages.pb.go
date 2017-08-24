// Code generated by protoc-gen-go. DO NOT EDIT.
// source: messages.proto

/*
Package messages is a generated protocol buffer package.

It is generated from these files:
	messages.proto

It has these top-level messages:
	PeerMessage
	HeartBeat
	PeerList
*/
package messages

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

// one size fit all message to just use one wire for a peer's message bus
type PeerMessage struct {
	HeartBeat        *HeartBeat `protobuf:"bytes,1,opt,name=heartBeat" json:"heartBeat,omitempty"`
	PeerList         *PeerList  `protobuf:"bytes,2,opt,name=peerList" json:"peerList,omitempty"`
	XXX_unrecognized []byte     `json:"-"`
}

func (m *PeerMessage) Reset()                    { *m = PeerMessage{} }
func (m *PeerMessage) String() string            { return proto.CompactTextString(m) }
func (*PeerMessage) ProtoMessage()               {}
func (*PeerMessage) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *PeerMessage) GetHeartBeat() *HeartBeat {
	if m != nil {
		return m.HeartBeat
	}
	return nil
}

func (m *PeerMessage) GetPeerList() *PeerList {
	if m != nil {
		return m.PeerList
	}
	return nil
}

type HeartBeat struct {
	CreatedTime      *uint64 `protobuf:"varint,1,req,name=createdTime" json:"createdTime,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *HeartBeat) Reset()                    { *m = HeartBeat{} }
func (m *HeartBeat) String() string            { return proto.CompactTextString(m) }
func (*HeartBeat) ProtoMessage()               {}
func (*HeartBeat) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *HeartBeat) GetCreatedTime() uint64 {
	if m != nil && m.CreatedTime != nil {
		return *m.CreatedTime
	}
	return 0
}

type PeerList struct {
	Addresses        []string `protobuf:"bytes,1,rep,name=addresses" json:"addresses,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *PeerList) Reset()                    { *m = PeerList{} }
func (m *PeerList) String() string            { return proto.CompactTextString(m) }
func (*PeerList) ProtoMessage()               {}
func (*PeerList) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

func (m *PeerList) GetAddresses() []string {
	if m != nil {
		return m.Addresses
	}
	return nil
}

func init() {
	proto.RegisterType((*PeerMessage)(nil), "PeerMessage")
	proto.RegisterType((*HeartBeat)(nil), "HeartBeat")
	proto.RegisterType((*PeerList)(nil), "PeerList")
}

func init() { proto.RegisterFile("messages.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 159 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x4c, 0xcc, 0x31, 0xcb, 0xc2, 0x30,
	0x10, 0xc6, 0x71, 0xd2, 0xf7, 0x1d, 0x9a, 0x2b, 0x38, 0x64, 0xca, 0xe0, 0x10, 0x0a, 0x42, 0x16,
	0x33, 0xf8, 0x11, 0x9c, 0x1c, 0x14, 0x4a, 0x70, 0x16, 0x82, 0x79, 0xd0, 0x0e, 0xa5, 0xe5, 0x2e,
	0xdf, 0x1f, 0x41, 0x6d, 0xeb, 0xfa, 0xfb, 0xdf, 0x3d, 0xb4, 0x19, 0x20, 0x92, 0x1e, 0x90, 0x30,
	0xf1, 0x58, 0xc6, 0xf6, 0x46, 0x4d, 0x07, 0xf0, 0xe5, 0xa3, 0xc6, 0x93, 0x7e, 0x22, 0x71, 0x39,
	0x22, 0x15, 0xab, 0x9c, 0xf2, 0xcd, 0x81, 0xc2, 0x69, 0x96, 0xb8, 0x46, 0xb3, 0xa3, 0x7a, 0x02,
	0xf8, 0xdc, 0x4b, 0xb1, 0xd5, 0xfb, 0x50, 0x87, 0xee, 0x0b, 0x71, 0x49, 0xed, 0x9e, 0xf4, 0xf2,
	0x6e, 0x1c, 0x35, 0x77, 0x46, 0x2a, 0xc8, 0xd7, 0x7e, 0x80, 0x55, 0xae, 0xf2, 0xff, 0xf1, 0x97,
	0x5a, 0x4f, 0xf5, 0x3c, 0x62, 0xb6, 0xa4, 0x53, 0xce, 0x0c, 0x11, 0x88, 0x55, 0xee, 0xcf, 0xeb,
	0xb8, 0xc2, 0x2b, 0x00, 0x00, 0xff, 0xff, 0x4a, 0x0f, 0xab, 0x8b, 0xc9, 0x00, 0x00, 0x00,
}
