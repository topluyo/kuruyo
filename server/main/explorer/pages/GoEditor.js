
function InitGoEditor(){
  require(['vs/editor/editor.main'], function () {
    monaco.editor.getEditors().map(editor=>editor.onDidBlurEditorWidget(() => {
      editor.tempValue = editor.getValue();
    }))
    function AllLines(){
      return monaco.editor.getEditors().map(e=>{
        if(e==fileManager.editor){
          return e.getValue()
        }
        return e._tempValue
      }).join("\n")
    }

    function detectStructFields(){
      let lines = AllLines().split("\n");
      let functions = [];
      lines.forEach((line) => {
        let match = line.match(/type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\s*/);
        if (match) {
          functions.push(match[1]);
        }
      });
      return functions;
    }

    function detectGoFunction(text) {
      let lines = AllLines().split("\n");
      let functions = [];
      lines.forEach((line) => {
        let match = line.match(/func\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(/);
        if (match) {
          functions.push(match[1]);
        }
      });
      return functions;
    }


    function detectAllGoFunctionParameters(text) {
      let lines = AllLines().split("\n");
      let functions = {};
      lines.forEach((line) => {
        let match = line.match(/func\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(/);
        if (match) {
          
          functions[match[1]] = [];
          let param =  line.match( /func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)/ );
          if (param) {
            functions[match[1]] = [...param[2].split(",").map(e=>e.trim().split(/\s+/).join(":") )];
          }
        }
      });
      return functions;
    }


    function detectGoCurrentFunctionParameters(text, index) {
      let lines = text.split("\n");
      let parameters = [];
      let functionFound = false;

      for ( let i = index; i >= 0; i-- ) {
        let line = lines[i];
        let tabIndent = line.match(/^\t*/)[0].length;
        let spaceIndent = line.match(/^ */)[0].length;
        let indent = tabIndent + spaceIndent/2;
        

        let match =  line.match( /func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)/ );
        if (match) {
          parameters.push(...match[2].split(",").map(e=>e.trim().split(" ")[0] ));
          break
        }

        if( indent==0 && line.trim()!="" ) break;
        

      }
      console.log(parameters);
      parameters = [...new Set(parameters)];
      return parameters;
    }

    function detectPythonVariables(text) {
      let lines = text.split("\n");
      let variables = [];
      lines.forEach((line) => {
        let match = line.match(/([a-zA-Z0-9_]*)\s*\:\=/);
        if (match) {
          variables.push(match[1]);
        }
        match = line.match(/var ([a-zA-Z0-9_]*)\s*/);
        if (match) {
          variables.push(match[1]);
        }
      });
      // unique variables
      variables = [...new Set(variables)];
      return variables;
    }




    function createDependencyProposals(range, text) {

      let staticFunctions = []

      let structFields = detectStructFields()
      let functions = detectGoFunction(text)
      let variables = detectPythonVariables(text)
      let parameters = detectGoCurrentFunctionParameters(text,range.startLineNumber-1)

      functions = functions.map(e => ({
        label: e,
        insertText: e , // +"()",
        range: range,
        kind: monaco.languages.CompletionItemKind.Function,
        sortText : "3",
        //detail:"Standart giriÅŸ Ã§Ä±kÄ±ÅŸ fonksiyonlarÄ±",
        //unit:"byte"
      }))

      variables = variables.map(e => ({
        label: e,
        insertText: e,
        range: range,
        kind: monaco.languages.CompletionItemKind.Variable,
        sortText : "2",
        //detail:"Standart giriÅŸ Ã§Ä±kÄ±ÅŸ fonksiyonlarÄ±",
        //unit:"byte"
      }))


      parameters = parameters.map(e => ({
        label: e,
        insertText: e,
        range: range,
        kind: monaco.languages.CompletionItemKind.Variable,
        sortText : "1",
        //detail:"Standart giriÅŸ Ã§Ä±kÄ±ÅŸ fonksiyonlarÄ±",
        //unit:"byte"
      }))



      structFields = structFields.map((e) => ({
        label: e,
        insertText: e,
        range: range,
        kind: monaco.languages.CompletionItemKind.Field,
        sortText: "4",
      }));


      let keywords = [
        "int","string","double",
        "if","else","for","return","func","var","const","struct","interface",
        "switch","case","defer","go","map","chan","package","import"] //,"import","from","as","class","try","except","finally","with","assert","global","nonlocal","lambda","del","yield","in","is","not","and","or","as","True","False","None"]
      keywords = keywords.map(e=>({
        label:e,
        insertText:e,
        range:range,
        kind: monaco.languages.CompletionItemKind.Keyword,
        documentaion:"Standart anahtar kelimeler",
        sortText:"9"
        //detail:"Standart giriÅŸ Ã§Ä±kÄ±ÅŸ fonksiyonlarÄ±",
        //unit:"byte"
      }))




      return [...functions, ...variables, ...parameters,...structFields,...keywords]

    }


      

    // 1️⃣ Detect all structs and their fields
    function detectGoStructs(text) {
      const lines = AllLines().split("\n");
      const structs = {};
      let currentStruct = null;

      for (let line of lines) {
        let structMatch = line.match(/type\s+([A-Za-z_][A-Za-z0-9_]*)\s+struct\s*{/);
        if (structMatch) {
          currentStruct = structMatch[1];
          structs[currentStruct] = [];
          continue;
        }

        if (currentStruct) {
          let fieldMatch = line.match(/^\s*([A-Za-z_][A-Za-z0-9_]*)\s+[A-Za-z0-9_\[\]*]+/);
          if (fieldMatch) structs[currentStruct].push(fieldMatch[1]);
          if (line.includes("}")) currentStruct = null;
        }
      }

      return structs; // { StructName: [field1, field2, ...] }
    }

    // 2️⃣ Detect variables and their types
    function detectGoVariableTypes(text) {
      const lines = text.split("\n");
      const variableTypes = {};

      lines.forEach(line => {
        // var x Type
        let varMatch = line.match(/var\s+([a-zA-Z_][a-zA-Z0-9_]*)\s+\**([A-Za-z_][A-Za-z0-9_]*)/);
        if (varMatch) {
          variableTypes[varMatch[1]] = varMatch[2];
          return;
        }

        // short declaration: x := Type{}
        let shortMatch = line.match(/([a-zA-Z_][a-zA-Z0-9_]*)\s*:=\s*([A-Za-z_][A-Za-z0-9_]*)\s*{}/);
        if (shortMatch) {
          variableTypes[shortMatch[1]] = shortMatch[2];
          return;
        }

        let shortMatchFn = line.match(/([a-zA-Z_][a-zA-Z0-9_]*)\s*:=\s*([A-Za-z_][A-Za-z0-9_]*)/);
        if (shortMatchFn) {
          if(shortMatchFn[2].startsWith("Model_")) shortMatchFn[2] = shortMatchFn[2].replace("Model_","Struct_Model_")
          if(shortMatchFn[2].startsWith("Func_")) shortMatchFn[2] = shortMatchFn[2].replace("Func_","Struct_Model_")
          if(shortMatchFn[2].startsWith("Get_")) shortMatchFn[2] = shortMatchFn[2].replace("Get_","Struct_Get_")
          variableTypes[shortMatchFn[1]] = shortMatchFn[2];
          return;
        }
      });

      return variableTypes; // { user: "User", u: "User" }
    }

    // 3️⃣ Get fields for a specific object
    function detectStructFieldsForObject(text, objectName) {
      const structs = detectGoStructs(text);
      const variableTypes = detectGoVariableTypes(text);

      const structName = variableTypes[objectName];
      if (structName && structs[structName]) {
        return structs[structName].map(field => ({
          label: field,
          insertText: field,
          kind: monaco.languages.CompletionItemKind.Field,
          sortText: "4"
        }));
      }

      return [];
    }



    monaco.languages.registerCompletionItemProvider('go', {
      triggerCharacters: ['.'],
      provideCompletionItems: function (model, position) {


        const lineContent = model.getLineContent(position.lineNumber);
        const objectMatch = lineContent.slice(0, position.column - 1).match(/([a-zA-Z_][a-zA-Z0-9_]*)\.[a-zA-Z0-9_]*$/);

        if (objectMatch) {
          const objectName = objectMatch[1];
          const suggestions = detectStructFieldsForObject(model.getValue(), objectName);
          return { suggestions };
        }
        
        var word = model.getWordUntilPosition(position);
        var range = {
          startLineNumber: position.lineNumber,
          endLineNumber: position.lineNumber,
          startColumn: word.startColumn,
          endColumn: word.endColumn
        };
        return {
          suggestions: createDependencyProposals(range, model.getValue())
        };
      },
    })



    monaco.languages.registerSignatureHelpProvider('go', {
      signatureHelpTriggerCharacters: ['(', ','],
      provideSignatureHelp: function (model, position) {
        const word = model.getWordUntilPosition(position);

        const lineContent = model.getLineContent(position.lineNumber);
        
        const textBeforeCursor = lineContent.substring(0, position.column - 1);
        const commaCount = (textBeforeCursor.match(/,/g) || []).length;

        const fnParameters = detectAllGoFunctionParameters(AllLines());
        //console.log(fnParameters)
        console.log(lineContent)
        for(let fn in fnParameters){
          let params = fnParameters[fn]
          console.log(lineContent)
          let founded = lineContent.split(/\W/).includes(fn)
          if (founded) {
            return {
              value: {
                signatures: [
                  {
                    documentation: fn + '('+params.map(e=>e.split(":")[0]).join(", ")+')',
                    label: params.map(e=>e.split(":").join(": ")).join(", "),
                    parameters:  params.map(e=>{
                      return {
                        label: e.split(":").join(": "),
                        documentation: e.split(":").join(": "),
                      }
                    }),
                  }
                ],
                activeSignature: 0,
                activeParameter: commaCount
              },
              dispose: () => {}
            };
          }
        }


        return { value: { signatures: [], activeSignature: 0, activeParameter: 0 }, dispose: () => {} };
      }
    });
  });
}


