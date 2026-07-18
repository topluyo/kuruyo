function TextToCsv(data) {
	const rows = [];
	let row = [];
	let field = "";
	const len = data.length;
	for (let i = 0; i < len; i++) {
		const c = data[i];
		if (c === "\\") {
				i++;
				if (i >= len) {
					field += "\\";
					break;
				}
				switch (data[i]) {
					case "\\":
						field += "\\";
						break;

					case ",":
						field += ",";
						break;

					case "n":
						field += "\n";
						break;

					case "r":
						field += "\r";
						break;

					default:
						// bilinmeyen escape
						field += "\\";
						field += data[i];
						break;
				}
				continue;
			}
			if (c === ",") {
				row.push(field);
				field = "";
				continue;
			}
      // Windows (\r\n) satır sonundaki \r karakterini atla
      if (c === "\r"){
        continue;
      }
			if (c === "\n") {
				row.push(field);
				rows.push(row);
				row = [];
				field = "";
				continue;
			}
			field += c;
	}
	// son alan
	if (field.length > 0 || row.length > 0) {
		row.push(field);
		rows.push(row);
	}
	return rows;
}


function CsvTextToJSON(data) {
  const result = [];
  let headers = null;
  let row = [];
  let field = "";
  const len = data.length;
  for (let i = 0; i < len; i++) {
    const c = data[i];
    // escape karakteri
    if (c === "\\") {
      i++;
      if (i >= len) {
          field += "\\";
          break;
      }
      switch (data[i]) {
        case "\\":
          field += "\\";
          break;

        case ",":
          field += ",";
          break;

        case "n":
          field += "\n";
          break;

        case "r":
          field += "\r";
          break;

        default:
          field += "\\";
          field += data[i];
          break;
      }
      continue;
    }

    // kolon
    if (c === ",") {
      row.push(field);
      field = "";
      continue;
    }

    // Windows (\r\n) satır sonundaki \r karakterini atla
    if (c === "\r"){
      continue;
    }

    // satır
    if (c === "\n") {
      row.push(field);
      field = "";
      if (headers === null) {
        headers = row;
      } else {
        const obj = {};
        const count = headers.length;
        for (let x = 0; x < count; x++) {
          obj[headers[x]] = row[x] ?? "";
        }
        result.push(obj);
      }
      row = [];
      continue;
    }
    field += c;
  }


  // son satır (\n ile bitmeyen CSV)
  if (field.length > 0 || row.length > 0) {
    row.push(field);
    if (headers === null) {
      headers = row;
    } else {
      const obj = {};
      for (let x = 0; x < headers.length; x++) {
        obj[headers[x]] = row[x] ?? "";
      }
      result.push(obj);
    }
  }
  return result;
}

