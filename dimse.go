package netdicom

// Implements message types defined in P3.7.
//
// http://dicom.nema.org/medical/dicom/current/output/pdf/part07.pdf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/yasushi-saito/go-dicom"
	"io"
	"log"
)

type DIMSEMessage interface {
	Encode(*dicom.Encoder)
	HasData() bool
	DebugString() string
}

func findElementWithTag(elems []*dicom.DicomElement, tag dicom.Tag) (*dicom.DicomElement, error) {
	for _, elem := range elems {
		if elem.Tag == tag {
			log.Printf("Return %v for %s", elem, tag.String())
			return elem, nil
		}
	}

	return nil, fmt.Errorf("Element %s not found during DIMSE decoding", tag.String())
}

func getStringFromElements(elems []*dicom.DicomElement, tag dicom.Tag) (string, error) {
	e, err := findElementWithTag(elems, tag)
	if err != nil {
		return "", err
	}
	return dicom.GetString(*e)
}

func getUInt32FromElements(elems []*dicom.DicomElement, tag dicom.Tag) (uint32, error) {
	e, err := findElementWithTag(elems, tag)
	if err != nil {
		return 0, err
	}
	return dicom.GetUInt32(*e)
}

func getUInt16FromElements(elems []*dicom.DicomElement, tag dicom.Tag) (uint16, error) {
	e, err := findElementWithTag(elems, tag)
	if err != nil {
		return 0, err
	}
	return dicom.GetUInt16(*e)
}

// Fields common to all DIMSE messages.
type DIMSEMessageHeader struct {
	CommandField uint16 // (0000,0100)
}

func encodeDataElementWithSingleValue(e *dicom.Encoder, tag dicom.Tag, v interface{}) {
	values := []interface{}{v}
	dicom.EncodeDataElement(e, tag, values)
}

func encodeDIMSEMessageHeader(e *dicom.Encoder, v DIMSEMessageHeader) {
	//encodeDataElementWithSingleValue(e, dicom.Tag{0, 0}, v.CommandGroupLength)
	//encodeDataElementWithSingleValue(e, dicom.Tag{0, 2}, v.AffectedSOPClassUID)
}

// Standard DIMSE tags
var (
	TagCommandGroupLength                   = dicom.Tag{0, 0}
	TagCommandField                         = dicom.Tag{0, 0x100}
	TagAffectedSOPClassUID                  = dicom.Tag{0x0000, 0x0002}
	TagMessageID                            = dicom.Tag{0000, 0x0110}
	TagMessageIDBeingRespondedTo            = dicom.Tag{0000, 0x0120}
	TagPriority                             = dicom.Tag{0000, 0x0700}
	TagCommandDataSetType                   = dicom.Tag{0000, 0x0800}
	TagStatus                               = dicom.Tag{0000, 0x0900}
	TagAffectedSOPInstanceUID               = dicom.Tag{0000, 0x1000}
	TagMoveOriginatorApplicationEntityTitle = dicom.Tag{0000, 0x1030}
	TagMoveOriginatorMessageID              = dicom.Tag{0000, 0x1031}
)

// P3.7 9.3.1.1
type C_STORE_RQ struct {
	AffectedSOPClassUID                  string
	MessageID                            uint16
	Priority                             uint16
	CommandDataSetType                   uint16
	AffectedSOPInstanceUID               string
	MoveOriginatorApplicationEntityTitle string
	MoveOriginatorMessageID              uint16
}

func (v *C_STORE_RQ) HasData() bool {
	doassert(v.CommandDataSetType != CommandDataSetTypeNull) // TODO(saito)
	return v.CommandDataSetType != CommandDataSetTypeNull
}

func (v *C_STORE_RQ) Encode(e *dicom.Encoder) {
	encodeDataElementWithSingleValue(e, TagCommandField, uint16(1))
	encodeDataElementWithSingleValue(e, TagAffectedSOPClassUID, v.AffectedSOPClassUID)
	encodeDataElementWithSingleValue(e, dicom.Tag{0, 0x110}, v.MessageID)
	encodeDataElementWithSingleValue(e, dicom.Tag{0, 0x700}, v.Priority)
	encodeDataElementWithSingleValue(e, dicom.Tag{0, 0x800}, v.CommandDataSetType)
	encodeDataElementWithSingleValue(e, TagAffectedSOPInstanceUID, v.AffectedSOPInstanceUID)
	if v.MoveOriginatorApplicationEntityTitle != "" {
		encodeDataElementWithSingleValue(e, dicom.Tag{0, 1030}, v.MoveOriginatorApplicationEntityTitle)
	}
	if v.MoveOriginatorMessageID != 0 {
		encodeDataElementWithSingleValue(e, dicom.Tag{0, 1031}, v.MoveOriginatorMessageID)
	}
}

func decodeC_STORE_RQ(elems []*dicom.DicomElement) (*C_STORE_RQ, error) {
	v := C_STORE_RQ{}
	var err error
	v.AffectedSOPClassUID, err = getStringFromElements(elems, TagAffectedSOPClassUID)
	if err != nil {
		return nil, err
	}
	v.MessageID, err = getUInt16FromElements(elems, TagMessageID)
	if err != nil {
		return nil, err
	}
	v.Priority, err = getUInt16FromElements(elems, TagPriority)
	if err != nil {
		return nil, err
	}
	v.CommandDataSetType, err = getUInt16FromElements(elems, TagCommandDataSetType)
	if err != nil {
		return nil, err
	}
	v.AffectedSOPInstanceUID, err = getStringFromElements(elems, TagAffectedSOPInstanceUID)
	if err != nil {
		return nil, err
	}
	v.MoveOriginatorApplicationEntityTitle, _ = getStringFromElements(elems, TagMoveOriginatorApplicationEntityTitle)
	v.MoveOriginatorMessageID, _ = getUInt16FromElements(elems, TagMoveOriginatorMessageID)
	return &v, nil
}

func (v *C_STORE_RQ) DebugString() string {
	return fmt.Sprintf("cstorerq{sopclass:%v messageid:%v pri: %v cmddatasettype: %v sopinstance: %v m0:%v m1:%v}",
		v.AffectedSOPClassUID, v.MessageID, v.Priority, v.CommandDataSetType, v.AffectedSOPInstanceUID,
		v.MoveOriginatorApplicationEntityTitle, v.MoveOriginatorMessageID)
}

const CommandDataSetTypeNull uint16 = 0x101

// P3.7 9.3.1.2
type C_STORE_RSP struct {
	AffectedSOPClassUID       string
	MessageIDBeingRespondedTo uint16
	// CommandDataSetType shall always be 0x0101; RSP has no dataset.
	CommandDataSetType     uint16
	AffectedSOPInstanceUID string
	Status                 uint16
}

// C_STORE_RSP status codes.
// P3.4 GG4-1
const (
	CStoreStatusOutOfResources              uint16 = 0xa700
	CStoreStatusDataSetDoesNotMatchSOPClass uint16 = 0xa900
	CStoreStatusCannotUnderstand            uint16 = 0xc000
)

// P3.7 C
func decodeC_STORE_RSP(elems []*dicom.DicomElement) (*C_STORE_RSP, error) {
	v := &C_STORE_RSP{}
	var err error
	v.AffectedSOPClassUID, err = getStringFromElements(elems, TagAffectedSOPClassUID)
	if err != nil {
		return nil, err
	}
	v.MessageIDBeingRespondedTo, err = getUInt16FromElements(elems, TagMessageIDBeingRespondedTo)
	if err != nil {
		return nil, err
	}
	v.Status, err = getUInt16FromElements(elems, TagStatus)
	if err != nil {
		return nil, err
	}
	v.CommandDataSetType, err = getUInt16FromElements(elems, TagCommandDataSetType)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (v *C_STORE_RSP) Encode(e *dicom.Encoder) {
	doassert(v.CommandDataSetType == 0x101)
	encodeDataElementWithSingleValue(e, TagCommandField, uint16(0x8001))
	encodeDataElementWithSingleValue(e, TagAffectedSOPClassUID, v.AffectedSOPClassUID)
	encodeDataElementWithSingleValue(e, TagMessageIDBeingRespondedTo, v.MessageIDBeingRespondedTo)
	encodeDataElementWithSingleValue(e, TagCommandDataSetType, v.CommandDataSetType)
	encodeDataElementWithSingleValue(e, TagAffectedSOPInstanceUID, v.AffectedSOPInstanceUID)
	encodeDataElementWithSingleValue(e, TagStatus, v.Status)
}

func (v *C_STORE_RSP) HasData() bool {
	doassert(v.CommandDataSetType == CommandDataSetTypeNull) // TODO(saito)
	return v.CommandDataSetType != CommandDataSetTypeNull
}

func (v *C_STORE_RSP) DebugString() string {
	return fmt.Sprintf("cstorersp{sopclass:%v messageid:%v cmddatasettype: %v sopinstance: %v status: 0x%v}",
		v.AffectedSOPClassUID, v.MessageIDBeingRespondedTo, v.CommandDataSetType, v.AffectedSOPInstanceUID,
		v.Status)
}

func DecodeDIMSEMessage(io io.Reader, limit int64) (DIMSEMessage, error) {
	var elems []*dicom.DicomElement
	// Note: DIMSE elements are always implicit LE.
	//
	// TODO(saito) make sure that's the case. Where the ref?
	d := dicom.NewDecoder(io, limit, binary.LittleEndian, true /*implicit*/)
	for d.Len() > 0 && d.Error() == nil {
		elem := dicom.ReadDataElement(d)
		elems = append(elems, elem)
	}
	if err := d.Finish(); err != nil {
		return nil, err
	}

	commandField, err := getUInt16FromElements(elems, TagCommandField)
	if err != nil {
		return nil, err
	}
	switch commandField {
	case 1:
		return decodeC_STORE_RQ(elems)
	case 0x8001:
		return decodeC_STORE_RSP(elems)
	}
	log.Panicf("Unknown DIMSE command 0x%x", commandField)
	return nil, err
}

func encodeDIMSEMessage(v DIMSEMessage) ([]byte, error) {
	subEncoder := dicom.NewEncoder(binary.LittleEndian)
	v.Encode(subEncoder)
	bytes, err := subEncoder.Finish()
	if err != nil {
		return nil, err
	}

	e := dicom.NewEncoder(binary.LittleEndian)
	encodeDataElementWithSingleValue(e, TagCommandGroupLength, uint32(len(bytes)))
	e.EncodeBytes(bytes)
	return e.Finish()
}

type dimseCommandAssembler struct {
	contextID      byte
	commandBytes        []byte
	command DIMSEMessage
	dataBytes           []byte
	readAllCommand bool

	readAllData    bool
}

func addPDataTF(a *dimseCommandAssembler, pdu *P_DATA_TF, contextIDMap *contextIDMap) (string, DIMSEMessage, []byte, error) {
	for _, item := range pdu.Items {
		if a.contextID == 0 {
			a.contextID = item.ContextID
		} else if a.contextID != item.ContextID {
			// TODO(saito) don't panic here.
			log.Panicf("Mixed context: %d %d", a.contextID, item.ContextID)
		}
		if item.Command {
			a.commandBytes = append(a.commandBytes, item.Value...)
			if item.Last {
				doassert(!a.readAllCommand)
				a.readAllCommand = true
			}
		} else {
			a.dataBytes = append(a.dataBytes, item.Value...)
			if item.Last {
				doassert(!a.readAllData)
				a.readAllData = true
			}
		}
	}
	if !a.readAllCommand {
		return "", nil, nil, nil
	}
	if a.command == nil {
		var err error
		a.command, err = DecodeDIMSEMessage(bytes.NewBuffer(a.commandBytes), int64(len(a.commandBytes)))
		if err != nil {
			return "", nil, nil, err
		}
	}
	if a.command.HasData() && !a.readAllData {
		return "", nil, nil, nil
	}
	syntaxName, err := contextIDToAbstractSyntaxName(contextIDMap, a.contextID)
	command := a.command
	dataBytes := a.dataBytes
	log.Printf("Read all data for syntax %s, command [%v], data %d bytes, err%v",
		dicom.UIDDebugString(syntaxName), command.DebugString(), len(a.dataBytes), err)
	*a = dimseCommandAssembler{}
	return syntaxName, command, dataBytes, nil
		// TODO(saito) Verify that there's no unread items after the last command&data.
}
