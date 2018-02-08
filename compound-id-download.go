package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	// "github.com/golang/protobuf/jsonpb"
)

type Record struct {
	ID                string             `json:"id,omitempty"`
	GeneName          string             `json:"gene_name,omitempty"`
	EntrezID          int32              `json:"entrez_id,omitempty"`
	DrugName          string             `json:"drug_name,omitempty"`
	ChemblID          string             `json:"chembl_id,omitempty"`
	Publications      []int32            `json:"publications,omitempty"`
	InteractionTypes  []string           `json:"interaction_types,omitempty"`
	Sources           []string           `json:"sources,omitempty"`
	Attributes        []Attribute        `json:"attributes,omitempty"`
	InteractionClaims []InteractionClaim `json:"interaction_claims,omitempty"`
}

type Attribute struct {
	Name    string   `json:"name,omitempty"`
	Value   string   `json:"value,omitempty"`
	Sources []string `json:"sources,omitempty"`
}

type InteractionClaim struct {
	Source          string      `json:"source,omitempty"`
	Drug            string      `json:"drug,omitempty"`
	Gene            string      `json:"gene,omitempty"`
	IntractionTypes []string    `json:"interaction_types,omitempty"`
	Attributes      []Attribute `json:"attributes,omitempty"`
}

// CompoundIDs represents a subset of mappings from:
// https://www.ebi.ac.uk/unichem/rest/src_compound_id/{compound_id}/{source_id}
//
// Sources described here:
// https://www.ebi.ac.uk/unichem/ucquery/listSources
type CompoundID struct {
	// source_id 1
	ChEMBL string `json:"chembl,omitempty"`
	// source_id 22
	PubChem string `json:"pubchem,omitempty"`
	// source_id 2
	DrugBank string `json:"drugbank,omitempty"`
	// source_id 7
	ChEBI string `json:"chebi,omitempty"`
}

func GetCompoundIDs(chemblID string) (*CompoundID, error) {
	compound := &CompoundID{ChEMBL: chemblID}

	// example: https://www.ebi.ac.uk/unichem/rest/src_compound_id/CHEMBL12/1
	tmplURL := "https://www.ebi.ac.uk/unichem/rest/src_compound_id/%s/1"
	resp, err := http.Get(fmt.Sprintf(tmplURL, chemblID))
	if err != nil {
		return compound, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return compound, err
	}
	if resp.StatusCode != 200 {
		return compound, fmt.Errorf("[STATUS CODE - %d]\t%s", resp.StatusCode, body)
	}
	idMap := []map[string]string{}
	err = json.Unmarshal(body, &idMap)
	if err != nil {
		return compound, err
	}

	// https://www.ebi.ac.uk/unichem/ucquery/listSources
	drugbank := "2"
	chebi := "7"
	pubchem := "22"
	for _, v := range idMap {
		switch v["src_id"] {
		case chebi:
			compound.ChEBI = v["src_compound_id"]
		case drugbank:
			compound.DrugBank = v["src_compound_id"]
		case pubchem:
			compound.PubChem = v["src_compound_id"]
		}
	}
	return compound, nil
}

func main() {
	inputFile := ""
	outputFile := ""
	flag.StringVar(&inputFile, "interactions", inputFile, "interactions file generated from dgidb-download.go")
	flag.StringVar(&outputFile, "output", outputFile, "output file path")
	flag.Parse()

	if inputFile == "" {
		fmt.Println("interactions file must be provided")
		os.Exit(1)
	}

	file, err := os.Open(inputFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var out io.WriteCloser
	if outputFile != "" {
		outputFile, err = filepath.Abs(outputFile)
		if err != nil {
			panic(err)
		}
		d := filepath.Dir(outputFile)
		err = os.MkdirAll(d, 0755)
		if err != nil {
			panic(err)
		}
		out, err = os.Create(outputFile)
		if err != nil {
			panic(err)
		}
	} else {
		out = os.Stdout
	}
	defer out.Close()

	writer := json.NewEncoder(out)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		interaction := Record{}
		err = json.Unmarshal(scanner.Bytes(), &interaction)
		if err != nil {
			panic(err)
		}
		cid, _ := GetCompoundIDs(interaction.ChemblID)
		err = writer.Encode(cid)
		if err != nil {
			panic(err)
		}
	}
}
