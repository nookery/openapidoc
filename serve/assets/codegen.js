/* openapidoc native UI - code sample generator
 *
 * Produces a ready to run request snippet in several languages from a
 * normalised operation plus the values the user typed into the parameter
 * form.
 */
(function (global) {
  "use strict";

  /* ---------- value preparation ---------- */

  /* Builds the concrete URL, headers and body for a request. */
  function buildRequest(operation, values) {
    var path = operation.path;
    var pathParams = [];
    var query = [];
    var headers = {};
    var cookies = [];

    var params = operation.parameters || [];
    for (var i = 0; i < params.length; i++) {
      var param = params[i];
      var value = values[param.name];
      if (value === undefined || value === null || value === "") continue;

      if (param.in === "path") {
        path = path.replace(new RegExp("\\{" + escapeRegExp(param.name) + "\\}", "g"), encodeURIComponent(value));
        pathParams.push({ name: param.name, value: value });
      } else if (param.in === "query") {
        query.push({ name: param.name, value: value });
      } else if (param.in === "header") {
        headers[param.name] = value;
      } else if (param.in === "cookie") {
        cookies.push({ name: param.name, value: value });
      }
    }

    var url = path;
    if (query.length) {
      var search = [];
      for (var q = 0; q < query.length; q++) {
        search.push(
          encodeURIComponent(query[q].name) + "=" + encodeURIComponent(query[q].value)
        );
      }
      url += (url.indexOf("?") === -1 ? "?" : "&") + search.join("&");
    }

    var body = null;
    if (operation.requestBody && values.__body) {
      body = values.__body;
    }

    return {
      url: url,
      method: operation.method,
      headers: headers,
      cookies: cookies,
      body: body,
      contentType: operation.requestBody
        ? operation.requestBody.contentType
        : "application/json",
    };
  }

  function escapeRegExp(text) {
    return text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  }

  /* Single quotes a value for languages that use them. */
  function single(value) {
    return "'" + String(value).replace(/'/g, "\\'") + "'";
  }

  /* Double quotes a value for languages that use them. */
  function double(value) {
    return '"' + String(value).replace(/\\/g, "\\\\").replace(/"/g, '\\"') + '"';
  }

  /* Renders a value as a JSON literal, indented for readability. */
  function jsonLiteral(value) {
    if (value === undefined || value === null) return "";
    try {
      return JSON.stringify(value, null, 2);
    } catch (e) {
      return "";
    }
  }

  /* ---------- generators ---------- */

  var generators = {};

  generators.curl = function (req) {
    var lines = ["curl -X " + req.method + " '" + req.url + "' \\"];
    var names = Object.keys(req.headers);
    for (var i = 0; i < names.length; i++) {
      lines.push("  -H '" + names[i] + ": " + req.headers[names[i]] + "' \\");
    }
    if (req.cookies.length) {
      var parts = [];
      for (var c = 0; c < req.cookies.length; c++) {
        parts.push(req.cookies[c].name + "=" + req.cookies[c].value);
      }
      lines.push("  -H 'Cookie: " + parts.join("; ") + "' \\");
    }
    if (req.body) {
      lines.push("  -H 'Content-Type: " + req.contentType + "' \\");
      lines.push("  -d '" + req.body.replace(/'/g, "'\\''") + "'");
    }
    return lines.join("\n");
  };

  generators.javascript = function (req) {
    var headers = {};
    for (var name in req.headers) {
      if (Object.prototype.hasOwnProperty.call(req.headers, name)) {
        headers[name] = req.headers[name];
      }
    }
    if (req.body) headers["Content-Type"] = req.contentType;

    var out =
      "const response = await fetch('" + req.url + "', {\n" +
      "  method: '" + req.method + "',\n";

    if (Object.keys(headers).length) {
      out += "  headers: " + jsonLiteral(headers) + ",\n";
    }
    if (req.body) {
      out += "  body: " + jsonLiteral(req.body) + ",\n";
    }
    out += "});\n\n";
    out += "const data = await response.json();\n";
    out += "console.log(data);";
    return out;
  };

  generators.node = function (req) {
    var out = "const response = await fetch('" + req.url + "', {\n";
    out += "  method: '" + req.method + "',\n";
    out += "  headers: {\n";
    var names = Object.keys(req.headers);
    for (var i = 0; i < names.length; i++) {
      out += "    '" + names[i] + "': '" + req.headers[names[i]] + "',\n";
    }
    if (req.body) {
      out += "    'Content-Type': '" + req.contentType + "',\n";
    }
    out += "  },\n";
    if (req.body) {
      out += "  body: JSON.stringify(" + jsonLiteral(req.body) + "),\n";
    }
    out += "});\n\n";
    out += "console.log(await response.json());";
    return out;
  };

  generators.python = function (req) {
    var out = "import requests\n\n";
    out += "url = " + single(req.url) + "\n\n";
    out += "headers = {\n";
    var names = Object.keys(req.headers);
    for (var i = 0; i < names.length; i++) {
      out += "    " + single(names[i]) + ": " + single(req.headers[names[i]]) + ",\n";
    }
    if (req.body) {
      out += "    'Content-Type': " + single(req.contentType) + ",\n";
    }
    out += "}\n\n";
    if (req.body) {
      out += "payload = " + jsonLiteral(req.body) + "\n\n";
      out += "response = requests.request(" + single(req.method.toLowerCase()) +
        ", url, headers=headers, json=payload)\n";
    } else {
      out += "response = requests.request(" + single(req.method.toLowerCase()) +
        ", url, headers=headers)\n";
    }
    out += "print(response.json())";
    return out;
  };

  generators.go = function (req) {
    var out = "package main\n\n";
    out += "import (\n\t\"bytes\"\n\t\"fmt\"\n\t\"io\"\n\t\"net/http\"\n";
    if (req.body) out += "\t\"strings\"\n";
    out += ")\n\n";
    out += "func main() {\n";
    out += "\turl := " + double(req.url) + "\n\n";

    if (req.body) {
      out += "\tpayload := strings.NewReader(" + jsonLiteral(req.body) + ")\n\n";
    }
    out += "\treq, err := http.NewRequest(" + double(req.method) + ", url, ";
    out += req.body ? "payload)\n" : "nil)\n";
    out += "\tif err != nil {\n\t\tpanic(err)\n\t}\n\n";

    var names = Object.keys(req.headers);
    if (names.length || req.body) {
      out += "\treq.Header.Set(\"Content-Type\", " + double(req.contentType) + ")\n";
      for (var i = 0; i < names.length; i++) {
        out += "\treq.Header.Set(" + double(names[i]) + ", " + double(req.headers[names[i]]) + ")\n";
      }
      out += "\n";
    }

    out += "\tres, err := http.DefaultClient.Do(req)\n";
    out += "\tif err != nil {\n\t\tpanic(err)\n\t}\n\tdefer res.Body.Close()\n\n";
    out += "\tbody, _ := io.ReadAll(res.Body)\n";
    out += "\tfmt.Println(string(body))\n";
    out += "}";
    return out;
  };

  generators.swift = function (req) {
    var out = "import Foundation\n\n";
    out += "let url = URL(string: \"" + req.url + "\")!\n";
    out += "var request = URLRequest(url: url)\n";
    out += "request.httpMethod = \"" + req.method + "\"\n\n";
    var names = Object.keys(req.headers);
    for (var i = 0; i < names.length; i++) {
      out += "request.setValue(\"" + req.headers[names[i]] + "\", forHTTPHeaderField: \"" + names[i] + "\")\n";
    }
    if (req.body) {
      out += "request.setValue(\"" + req.contentType + "\", forHTTPHeaderField: \"Content-Type\")\n";
      out += "request.httpBody = \"" + req.body.replace(/"/g, '\\"') + "\".data(using: .utf8)\n";
    }
    out += "\nlet (data, _) = try await URLSession.shared.data(for: request)\n";
    out += "print(String(data: data, encoding: .utf8) ?? \"\")";
    return out;
  };

  generators.java = function (req) {
    var out = "HttpRequest request = HttpRequest.newBuilder()\n";
    out += "    .uri(URI.create(\"" + req.url + "\"))\n";
    out += "    .method(\"" + req.method + "\", ";
    out += req.body ? "HttpRequest.BodyPublishers.ofString(\"" + req.body.replace(/"/g, '\\"') + "\")" : "HttpRequest.BodyPublishers.noBody()";
    out += ")\n";
    var names = Object.keys(req.headers);
    for (var i = 0; i < names.length; i++) {
      out += "    .header(\"" + names[i] + "\", \"" + req.headers[names[i]] + "\")\n";
    }
    if (req.body) {
      out += "    .header(\"Content-Type\", \"" + req.contentType + "\")\n";
    }
    out += "    .build();\n\n";
    out += "HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());\n";
    out += "System.out.println(response.body());";
    return out;
  };

  generators.kotlin = function (req) {
    var out = "val request = Request.Builder()\n";
    out += "    .url(\"" + req.url + "\")\n";
    out += "    .method(\"" + req.method + "\", ";
    out += req.body ? "RequestBody.create(\"" + req.body.replace(/"/g, '\\"') + "\")" : "null";
    out += ")\n";
    var names = Object.keys(req.headers);
    for (var i = 0; i < names.length; i++) {
      out += "    .header(\"" + names[i] + "\", \"" + req.headers[names[i]] + "\")\n";
    }
    if (req.body) {
      out += "    .header(\"Content-Type\", \"" + req.contentType + "\")\n";
    }
    out += "    .build()\n\n";
    out += "client.newCall(request).execute().use { response ->\n";
    out += "    println(response.body?.string())\n";
    out += "}";
    return out;
  };

  /* ---------- public API ---------- */

  global.CodeSamples = {
    /* The languages offered in the code panel. */
    languages: [
      { id: "curl", name: "cURL" },
      { id: "javascript", name: "JavaScript" },
      { id: "python", name: "Python" },
      { id: "go", name: "Go" },
      { id: "node", name: "Node.js" },
      { id: "swift", name: "Swift" },
      { id: "java", name: "Java" },
      { id: "kotlin", name: "Kotlin" },
    ],

    /* Builds a request object from an operation and the form values. */
    buildRequest: buildRequest,

    /* Generates a snippet in the named language. */
    generate: function (language, operation, values) {
      var generator = generators[language];
      if (!generator) return "";
      return generator(buildRequest(operation, values));
    },
  };
})(window);
