// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package compiler

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/matthieu/go-ethereum/common"
)

const (
	supportedSolcVersion = "0.3.5"
	testSource           = `
contract test {
   /// @notice Will multiply ` + "`a`" + ` by 7.
   function multiply(uint a) returns(uint d) {
       return a * 7;
   }
}
`
	testCode = "0x6060604052602a8060106000396000f3606060405260e060020a6000350463c6888fa18114601a575b005b6007600435026060908152602090f3"
	testInfo = `{"source":"\ncontract test {\n   /// @notice Will multiply ` + "`a`" + ` by 7.\n   function multiply(uint a) returns(uint d) {\n       return a * 7;\n   }\n}\n","language":"Solidity","languageVersion":"0.1.1","compilerVersion":"0.1.1","compilerOptions":"--binary file --json-abi file --natspec-user file --natspec-dev file --add-std 1","abiDefinition":[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}],"userDoc":{"methods":{"multiply(uint256)":{"notice":"Will multiply ` + "`a`" + ` by 7."}}},"developerDoc":{"methods":{}}}`
)

func skipUnsupported(t *testing.T) {
	sol, err := SolidityVersion("")
	if err != nil {
		t.Skip(err)
		return
	}
	if sol.Version != supportedSolcVersion {
		t.Skipf("unsupported version of solc found (%v, expect %v)", sol.Version, supportedSolcVersion)
	}
}

func TestCompiler(t *testing.T) {
	skipUnsupported(t)
	contracts, err := CompileSolidityString("", testSource)
	if err != nil {
		t.Fatalf("error compiling source. result %v: %v", contracts, err)
	}
	if len(contracts) != 1 {
		t.Errorf("one contract expected, got %d", len(contracts))
	}
	c, ok := contracts["test"]
	if !ok {
		t.Fatal("info for contract 'test' not present in result")
	}
	if c.Code != testCode {
		t.Errorf("wrong code: expected\n%s, got\n%s", testCode, c.Code)
	}
	if c.Info.Source != testSource {
		t.Error("wrong source")
	}
	if c.Info.CompilerVersion != supportedSolcVersion {
		t.Errorf("wrong version: expected %q, got %q", supportedSolcVersion, c.Info.CompilerVersion)
	}
}

func TestCompileError(t *testing.T) {
	skipUnsupported(t)
	contracts, err := CompileSolidityString("", testSource[4:])
	if err == nil {
		t.Errorf("error expected compiling source. got none. result %v", contracts)
	}
	t.Logf("error: %v", err)
}

func TestSaveInfo(t *testing.T) {
	var cinfo ContractInfo
	err := json.Unmarshal([]byte(testInfo), &cinfo)
	if err != nil {
		t.Errorf("%v", err)
	}
	filename := path.Join(os.TempDir(), "solctest.info.json")
	os.Remove(filename)
	cinfohash, err := SaveInfo(&cinfo, filename)
	if err != nil {
		t.Errorf("error extracting info: %v", err)
	}
	got, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("error reading '%v': %v", filename, err)
	}
	if string(got) != testInfo {
		t.Errorf("incorrect info.json extracted, expected:\n%s\ngot\n%s", testInfo, string(got))
	}
	wantHash := common.HexToHash("0x9f3803735e7f16120c5a140ab3f02121fd3533a9655c69b33a10e78752cc49b0")
	if cinfohash != wantHash {
		t.Errorf("content hash for info is incorrect. expected %v, got %v", wantHash.Hex(), cinfohash.Hex())
	}
}
