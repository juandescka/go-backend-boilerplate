package labels

var FieldLabels = map[string]string{
	"Affirmation":           "Affirmasi",
	"AffirmationType":       "Jenis Affirmasi",
	"CreatedAt":             "Dibuat oleh",
	"Example":               "example",
	"FamilyCard":            "Kartu Keluarga",
	"FamilyPhoto":           "Foto Keluarga",
	"GPA":                   "Rata-Rata Nilai Rapor",
	"MaterialType":          "Tipe materi",
	"Name":                  "Nama",
	"NationalAcademicScore": "Nilai TKA",
	"ParentTransferType":    "Jenis Pindah Orang Tua",
	"PermissionIDs":         "Permission IDs",
	"StatementLetter":       "Surat Keterangan",

	"teacher":                "Guru",
	"government":             "Pemerintah",
	"military":               "TNI",
	"police":                 "Polri",
	"state_owned_enterprise": "BUMN",
	"private_company":        "Perusahaan Swasta",

	"pip":        "PIP",
	"dtsen":      "DTSEN",
	"disability": "Disabilitas",

	"family_photo":            "Foto Keluarga",
	"family_card":             "Kartu Keluarga",
	"gpa":                     "Rata-Rata Nilai Rapor",
	"national_academic_score": "Nilai TKA",
	"statement_letter":        "Surat Keterangan",
}

func Get(field string) string {
	if label, ok := FieldLabels[field]; ok {
		return label
	}
	return field
}
