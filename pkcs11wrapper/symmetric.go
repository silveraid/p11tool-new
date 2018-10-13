package pkcs11wrapper

import (
	"bytes"
	"encoding/hex"
	"github.com/miekg/pkcs11"
	"github.com/pkg/errors"
	"os"
	"strconv"
)

func (p11w *Pkcs11Wrapper) CreateSymKey(objectLabel string, keyLen int, keyType string) (aesKey pkcs11.ObjectHandle, err error) {

	var pkcs11_mech *pkcs11.Mechanism
	switch keyType {
	case "AES":
		pkcs11_mech = pkcs11.NewMechanism(pkcs11.CKM_AES_KEY_GEN, nil)
	case "GENERIC_SECRET":
		pkcs11_mech = pkcs11.NewMechanism(pkcs11.CKM_GENERIC_SECRET_KEY_GEN, nil)
	case "DES3":
		pkcs11_mech = pkcs11.NewMechanism(pkcs11.CKM_DES3_KEY_GEN, nil)
	case "SHA256_HMAC":
		if CaseInsensitiveContains(p11w.Library.Info.ManufacturerID, "ncipher") {
			pkcs11_mech = pkcs11.NewMechanism(pkcs11.CKM_NC_SHA256_HMAC_KEY_GEN, nil)
		} else {
			pkcs11_mech = pkcs11.NewMechanism(pkcs11.CKM_GENERIC_SECRET_KEY_GEN, nil)
		}
	case "SHA384_HMAC":
		if CaseInsensitiveContains(p11w.Library.Info.ManufacturerID, "ncipher") {
			pkcs11_mech = pkcs11.NewMechanism(pkcs11.CKM_NC_SHA384_HMAC_KEY_GEN, nil)
		} else {
			pkcs11_mech = pkcs11.NewMechanism(pkcs11.CKM_GENERIC_SECRET_KEY_GEN, nil)
		}
	default:
		pkcs11_mech = pkcs11.NewMechanism(pkcs11.CKM_GENERIC_SECRET_KEY_GEN, nil)
	}
	// default mech CKM_AES_KEY_GEN
	//pkcs11_mech := pkcs11.NewMechanism(pkcs11.CKM_AES_KEY_GEN, nil)

	// Overrides env first, then autodetect from vendor
	//switch {
	//case CaseInsensitiveContains(os.Getenv("SECURITY_PROVIDER_CONFIG_MECH"), "CKM_GENERIC_SECRET_KEY_GEN"):
	//       pkcs11_mech = pkcs11.NewMechanism(pkcs11.CKM_GENERIC_SECRET_KEY_GEN, nil)
	//case CaseInsensitiveContains(p11w.Library.Info.ManufacturerID, "SafeNet"):
	//       pkcs11_mech = pkcs11.NewMechanism(pkcs11.CKM_GENERIC_SECRET_KEY_GEN, nil)
	//}

	// get the required attributes
	requiredAttributes := p11w.GetSymPkcs11Template(objectLabel, keyLen, keyType)

	// generate the aes key
	aesKey, err = p11w.Context.GenerateKey(
		p11w.Session,
		[]*pkcs11.Mechanism{
			// vendor specific
			pkcs11_mech,
		},
		requiredAttributes,
	)
	if err != nil {
		ExitWithMessage("GenerateKey", err)
	}

	return
}

/* return a set of attributes that we require for our aes key */
func (p11w *Pkcs11Wrapper) GetSymPkcs11Template(objectLabel string, keyLen int,
	keyType string) (SymPkcs11Template []*pkcs11.Attribute) {

	// default CKA_KEY_TYPE
	var pkcs11VendorAttr []*pkcs11.Attribute
	var pkcs11KeyType    []*pkcs11.Attribute

	// Overrides Key Length
	SymKeyLength := keyLen
	pkcs11_keylen := os.Getenv("SECURITY_PROVIDER_CONFIG_KLEN")
	if len(pkcs11_keylen) > 0 {
		KeyLength, err := strconv.Atoi(pkcs11_keylen)
		if err != nil {
			return
		}
		SymKeyLength = KeyLength
	}

	switch keyType {

	//
	case "AES":
		pkcs11KeyType = []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_AES),
		}
		pkcs11VendorAttr = []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_DECRYPT, true),
			pkcs11.NewAttribute(pkcs11.CKA_ENCRYPT, true),
			pkcs11.NewAttribute(pkcs11.CKA_VALUE_LEN, SymKeyLength), /* KeyLength */
		}	
		pkcs11VendorAttr = append(pkcs11VendorAttr, pkcs11KeyType...)

	//
	case "DES3":

		pkcs11KeyType = []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_DES3),
		}
		if SymKeyLength != 0 { //CloudHSM does not set keyLen on DES3...
			pkcs11VendorAttr = []*pkcs11.Attribute{
				pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
				pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
				pkcs11.NewAttribute(pkcs11.CKA_DECRYPT, true),
				pkcs11.NewAttribute(pkcs11.CKA_ENCRYPT, true),
				pkcs11.NewAttribute(pkcs11.CKA_WRAP, true),
				pkcs11.NewAttribute(pkcs11.CKA_UNWRAP, true),
				pkcs11.NewAttribute(pkcs11.CKA_VALUE_LEN, SymKeyLength), /* KeyLength */
			}
		} else {
			pkcs11VendorAttr = []*pkcs11.Attribute{
				pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
				pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
				pkcs11.NewAttribute(pkcs11.CKA_DECRYPT, true),
				pkcs11.NewAttribute(pkcs11.CKA_ENCRYPT, true),
				pkcs11.NewAttribute(pkcs11.CKA_WRAP, true),
				pkcs11.NewAttribute(pkcs11.CKA_UNWRAP, true),				
			}	
		}
		pkcs11VendorAttr = append(pkcs11VendorAttr, pkcs11KeyType...)

	//
	case "GENERIC_SECRET":
		pkcs11KeyType = []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_GENERIC_SECRET),
		}
		pkcs11VendorAttr = []*pkcs11.Attribute{
		        pkcs11.NewAttribute(pkcs11.CKA_VALUE_LEN, SymKeyLength), /* KeyLength */
			pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
			pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
		}
		pkcs11VendorAttr = append(pkcs11VendorAttr, pkcs11KeyType...)

	//
	case "SHA256_HMAC":
		if CaseInsensitiveContains(p11w.Library.Info.ManufacturerID, "ncipher") {
			pkcs11KeyType = []*pkcs11.Attribute{
				pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_SHA256_HMAC),
			}
		} else {
			pkcs11KeyType = []*pkcs11.Attribute{
				pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_GENERIC_SECRET),
			}
		}
		pkcs11VendorAttr = []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_VALUE_LEN, SymKeyLength), /* KeyLength */
			pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
			pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
		}
		pkcs11VendorAttr = append(pkcs11VendorAttr, pkcs11KeyType...)

	//
	case "SHA384_HMAC":
		if CaseInsensitiveContains(p11w.Library.Info.ManufacturerID, "ncipher") {
			pkcs11KeyType = []*pkcs11.Attribute{
				pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_SHA384_HMAC),
			}
		} else {
			pkcs11KeyType = []*pkcs11.Attribute{
				pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_GENERIC_SECRET),
			}
		}
		pkcs11VendorAttr = []*pkcs11.Attribute{
		        pkcs11.NewAttribute(pkcs11.CKA_VALUE_LEN, SymKeyLength), /* KeyLength */
			pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
			pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
		}
		pkcs11VendorAttr = append(pkcs11VendorAttr, pkcs11KeyType...)

	//
	default:
		pkcs11KeyType = []*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_GENERIC_SECRET),
		}
		pkcs11VendorAttr = []*pkcs11.Attribute{
		        pkcs11.NewAttribute(pkcs11.CKA_VALUE_LEN, SymKeyLength), /* KeyLength */
			pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
			pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
		}
		pkcs11VendorAttr = append(pkcs11VendorAttr, pkcs11KeyType...)
	}

	// Overrides env first, then autodetect from vendor
	//switch {
	//case CaseInsensitiveContains(os.Getenv("SECURITY_PROVIDER_CONFIG_KEYTYPE"), "CKK_GENERIC_SECRET"):
	//       pkcs11_keytype = pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_GENERIC_SECRET)
	//case CaseInsensitiveContains(p11w.Library.Info.ManufacturerID, "softhsm") &&
	//      p11w.Library.Info.LibraryVersion.Major > 1 &&
	//     p11w.Library.Info.LibraryVersion.Minor > 2:
	// matches softhsm versions greater than 2.2 (scott patched 2.3)
	//      pkcs11_keytype = pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_GENERIC_SECRET)
	//case CaseInsensitiveContains(p11w.Library.Info.ManufacturerID, "ncipher"):
	//      pkcs11_keytype = pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_SHA256_HMAC)
	//case CaseInsensitiveContains(p11w.Library.Info.ManufacturerID, "SafeNet"):
	//      pkcs11_keytype = pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_GENERIC_SECRET)
	//}

	// Scott's Reference
	// default template common to all manufactures

	SymPkcs11Template = []*pkcs11.Attribute{
		// common to all
		pkcs11.NewAttribute(pkcs11.CKA_LABEL, objectLabel),      /* Name of Key */
		pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),             /* This key should persist */
		//pkcs11.NewAttribute(pkcs11.CKA_VALUE_LEN, SymKeyLength), /* KeyLength */
		pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
		// vendor specific override
	}
	SymPkcs11Template = append(SymPkcs11Template, pkcs11VendorAttr...)

	return
}

/* test CKM_SHA384_HMAC signing */
func (p11w *Pkcs11Wrapper) SignHmacSha384(o pkcs11.ObjectHandle, message []byte) (hmac []byte, err error) {

	// start the signing
	err = p11w.Context.SignInit(
		p11w.Session,
		[]*pkcs11.Mechanism{
			pkcs11.NewMechanism(pkcs11.CKM_SHA384_HMAC, nil),
		},
		o,
	)
	if err != nil {
		return
	}

	// do the signing
	hmac, err = p11w.Context.Sign(p11w.Session, message)
	if err != nil {
		return
	}

	return
}

// EncAESGCM test CKM_AES_GCM for encryption
func (p11w *Pkcs11Wrapper) EncAESGCM(o pkcs11.ObjectHandle, message []byte) (enc []byte, IV []byte,  err error) {

	//gcparams := pkcs11.NewGCMParams(make([]byte, 16), nil, 128)
	gcparams := pkcs11.NewGCMParams([]byte{}, nil, 128)
	err = p11w.Context.EncryptInit(
		p11w.Session,
		[]*pkcs11.Mechanism{
			pkcs11.NewMechanism(pkcs11.CKM_AES_GCM, gcparams),
		},
		o,
	)
	if err != nil {
		return
	}

	// do the encryption
	enc, err = p11w.Context.Encrypt(p11w.Session, message) 
	if err != nil {
		return
	}
	//IV = enc[0:16]
	//IV is appended to End in Gemalto
	IV = enc[len(enc)-16:]
	result := bytes.Join([][]byte{gcparams.IV(), enc}, nil)
	//the above results in the join as equal to enc as IV() returns nil for Gemalto Testing
	gcparams.Free()

	return result, IV, nil
}

// DecAESGCM test CKM_AES_GCM for Decryption
func (p11w *Pkcs11Wrapper) DecAESGCM(o pkcs11.ObjectHandle, cipherText []byte, IV []byte) (message []byte, err error) {

	// gcparams := pkcs11.NewGCMParams(make([]byte, 16), nil, 128)
	gcparams := pkcs11.NewGCMParams(IV, nil, 128)
	err = p11w.Context.DecryptInit(
		p11w.Session,
		[]*pkcs11.Mechanism{
                        pkcs11.NewMechanism(pkcs11.CKM_AES_GCM, gcparams),
                },
		o,
	)
	if err != nil {
		return nil, err
	}
	cipherText = cipherText[:len(cipherText)-16]
	message, err = p11w.Context.Decrypt(p11w.Session, cipherText)
	if err != nil {
		return nil, err
	}
	gcparams.Free()

	return
}

//
func (p11w *Pkcs11Wrapper) ImportSymmetricKey(objectLabel, keyType, keyValue string) (error) {

	// input validation
	if objectLabel == "" || keyType == "" || keyValue == "" {
		return errors.New("FATAL: ImportSymmetricKey input validation failed")
	}

	// check if the key length is okay
	if len(keyValue) % 2 != 0 {
		return errors.Errorf("FATAL: ImportSymmetricKey key length invalid (%v)",
			len(keyValue))
	}

	// generating a key template
	keyTemplate := p11w.GetSymPkcs11Template(objectLabel, len(keyValue) / 2, keyType)

	// converting ASCII HEX string to byte array
	keyValueBytes, err := hex.DecodeString(keyValue)

	if err != nil {
		return errors.Errorf("FATAL: ImportSymmetricKey unable to convert " +
			"keyValue to byte array, %v", err)
	}

	// extending the template with the key value
	keyTemplate = append(keyTemplate, pkcs11.NewAttribute(pkcs11.CKA_VALUE, keyValueBytes))

	// trying to create the object in the HSM
	_, err = p11w.Context.CreateObject(p11w.Session, keyTemplate)

	// problem
	if err != nil {
		return errors.Errorf("FATAL: ImportSymmetricKey, %v", err)
	}

	// no problem
	return nil
}