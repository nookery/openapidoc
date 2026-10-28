/* openapidoc native UI - application
 *
 * Renders the OpenAPI document and drives the interactions: navigation,
 * search, parameter entry, schema trees, code samples and live requests.
 */
(function () {
  "use strict";

  var state = {
    model: null,
    operation: null,
    groups: [],
    activeGroup: null,
    query: "",
    language: "curl",
    values: {},
    collapsed: {},
    codeOpen: true,
    specUrl: "",
    title: "",
  };

  var el = {};

  /* ---------- utilities ---------- */

  function $(selector, root) {
    return (root || document).querySelector(selector);
  }

  function create(tag, className, text) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined && text !== null) node.textContent = String(text);
    return node;
  }

  function clear(node) {
    while (node.firstChild) node.removeChild(node.firstChild);
  }

  /* Escapes text for safe insertion into innerHTML. */
  function escapeHtml(text) {
    return String(text)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  /* Renders a minimal subset of markdown as HTML. */
  function renderMarkdown(text) {
    if (!text) return "";
    var html = escapeHtml(text);

    html = html.replace(/`([^`]+)`/g, "<code>$1</code>");
    html = html.replace(
      /\*\*([^*]+)\*\*/g,
      "<strong>$1</strong>"
    );
    html = html.replace(
      /(^|[^*])\*([^*\n]+)\*/g,
      "$1<em>$2</em>"
    );
    html = html.replace(
      /\[([^\]]+)\]\(([^)]+)\)/g,
      '<a href="$2" target="_blank" rel="noreferrer">$1</a>'
    );

    var lines = html.split("\n");
    var out = [];
    var inList = false;
    for (var i = 0; i < lines.length; i++) {
      var line = lines[i];
      var bullet = line.match(/^\s*[-*+]\s+(.*)$/);
      var ordered = line.match(/^\s*\d+\.\s+(.*)$/);

      if (bullet) {
        if (!inList) {
          out.push("<ul>");
          inList = true;
        }
        out.push("<li>" + bullet[1] + "</li>");
        continue;
      }
      if (inList) {
        out.push("</ul>");
        inList = false;
      }
      if (!line.trim()) continue;
      out.push("<p>" + line + "</p>");
    }
    if (inList) out.push("</ul>");
    return out.join("");
  }

  function showToast(message) {
    if (!el.toast) return;
    el.toast.textContent = message;
    el.toast.classList.add("is-visible");
    clearTimeout(showToast.timer);
    showToast.timer = setTimeout(function () {
      el.toast.classList.remove("is-visible");
    }, 2000);
  }

  /* ---------- syntax highlighting ---------- */

  var KEYWORDS = {
    curl: ["curl"],
    javascript: ["const", "let", "var", "await", "async", "function", "return", "new", "import", "from", "export", "if", "else", "for", "of", "in", "true", "false", "null", "undefined", "try", "catch", "throw"],
    python: ["import", "from", "as", "def", "return", "if", "elif", "else", "for", "in", "while", "try", "except", "class", "with", "print", "True", "False", "None", "and", "or", "not", "lambda", "await", "async"],
    go: ["package", "import", "func", "return", "if", "else", "for", "range", "var", "const", "type", "struct", "interface", "defer", "go", "chan", "map", "nil", "true", "false", "err", "string", "int"],
    node: ["const", "let", "var", "await", "async", "function", "return", "new", "import", "from", "export", "if", "else", "for", "of", "true", "false", "null", "try", "catch", "throw"],
    swift: ["import", "func", "return", "if", "else", "for", "in", "let", "var", "class", "struct", "enum", "protocol", "extension", "guard", "do", "try", "catch", "throw", "nil", "true", "false", "await", "async"],
    java: ["public", "private", "class", "static", "void", "new", "return", "if", "else", "for", "while", "try", "catch", "throw", "import", "package", "final", "int", "String", "boolean", "true", "false", "null"],
    kotlin: ["fun", "val", "var", "return", "if", "else", "for", "while", "class", "object", "interface", "import", "package", "try", "catch", "throw", "null", "true", "false", "buildString"],
  };

  /* Highlights a snippet as HTML. */
  function highlight(code, language) {
    if (!code) return "";
    var keywords = KEYWORDS[language] || [];
    var pattern = keywords
      .map(function (word) {
        return word.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
      })
      .join("|");

    var token =
      /(\/\/[^\n]*|#[^\n]*|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|`(?:[^`\\]|\\.)*`|\b\d+(?:\.\d+)?\b|\b(?:true|false|null|None|True|False|undefined)\b|\b(?:KEYWORDS)\b)/g;

    var keywordSet = {};
    for (var i = 0; i < keywords.length; i++) keywordSet[keywords[i]] = true;

    var out = "";
    var last = 0;
    var match;
    var regex = new RegExp(
      pattern
        ? "(\\/\\/[^\\n]*|#[^\\n]*|\"(?:[^\"\\\\]|\\\\.)*\"|'(?:[^'\\\\]|\\\\.)*'|`(?:[^`\\\\]|\\\\.)*`|\\b\\d+(?:\\.\\d+)?\\b|\\b(?:true|false|null|None|True|False|undefined)\\b|\\b(?:" +
          pattern +
          ")\\b)"
        : "(\\/\\/[^\\n]*|#[^\\n]*|\"(?:[^\"\\\\]|\\\\.)*\"|'(?:[^'\\\\]|\\\\.)*'|`(?:[^`\\\\]|\\\\.)*`|\\b\\d+(?:\\.\\d+)?\\b|\\b(?:true|false|null|None|True|False|undefined)\\b)",
      "g"
    );

    while ((match = regex.exec(code)) !== null) {
      out += escapeHtml(code.slice(last, match.index));
      var text = match[0];
      var cls = "tok-punc";
      if (/^(\/\/|#)/.test(text)) cls = "tok-comment";
      else if (/^["'`]/.test(text)) cls = "tok-str";
      else if (/^\d/.test(text)) cls = "tok-num";
      else if (/^(true|false|True|False)$/.test(text)) cls = "tok-bool";
      else if (/^(null|None|undefined)$/.test(text)) cls = "tok-null";
      else if (keywordSet[text]) cls = "tok-key";
      out += '<span class="' + cls + '">' + escapeHtml(text) + "</span>";
      last = match.index + text.length;
    }
    out += escapeHtml(code.slice(last));
    return out;
  }

  /* ---------- sidebar ---------- */

  function renderSidebar() {
    var nav = el.nav;
    clear(nav);

    var query = state.query.trim().toLowerCase();

    for (var g = 0; g < state.groups.length; g++) {
      var group = state.groups[g];
      var matched = [];

      for (var o = 0; o < group.operations.length; o++) {
        var op = group.operations[o];
        if (!query || operationMatches(op, query)) matched.push(op);
      }

      if (query && !matched.length) continue;

      var groupEl = create("div", "nav-group");
      var isCollapsed = state.collapsed["group:" + group.name];
      if (isCollapsed) groupEl.classList.add("is-collapsed");

      var title = create("button", "nav-group-title");
      title.type = "button";
      title.innerHTML =
        '<svg class="chevron" viewBox="0 0 24 24" fill="none" stroke="currentColor" ' +
        'stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">' +
        '<polyline points="6 9 12 15 18 9"></polyline></svg>';
      title.appendChild(document.createTextNode(group.name));
      var count = create("span", "nav-group-count", matched.length);
      title.appendChild(count);

      (function (groupName, groupEl) {
        title.addEventListener("click", function () {
          var key = "group:" + groupName;
          state.collapsed[key] = !state.collapsed[key];
          groupEl.classList.toggle("is-collapsed");
        });
      })(group.name, groupEl);

      groupEl.appendChild(title);

      var items = create("div", "nav-group-items");
      for (var m = 0; m < matched.length; m++) {
        items.appendChild(buildNavItem(matched[m]));
      }
      groupEl.appendChild(items);
      nav.appendChild(groupEl);
    }

    if (!nav.firstChild) {
      var empty = create("div", "empty");
      empty.appendChild(create("div", "empty-title", "没有匹配的接口"));
      empty.appendChild(
        create("div", "empty-hint", "换一个关键词试试，或者清空搜索框。")
      );
      nav.appendChild(empty);
    }
  }

  function operationMatches(op, query) {
    if (op.path.toLowerCase().indexOf(query) !== -1) return true;
    if (op.method.toLowerCase().indexOf(query) !== -1) return true;
    if (op.summary && op.summary.toLowerCase().indexOf(query) !== -1) return true;
    if (op.operationId && op.operationId.toLowerCase().indexOf(query) !== -1) {
      return true;
    }
    for (var i = 0; i < op.tags.length; i++) {
      if (op.tags[i].toLowerCase().indexOf(query) !== -1) return true;
    }
    return false;
  }

  function buildNavItem(op) {
    var item = create("button", "nav-item");
    item.type = "button";
    if (state.operation && state.operation.id === op.id) {
      item.classList.add("is-active");
    }

    var method = create("span", "method method-" + op.method.toLowerCase(), op.method);
    item.appendChild(method);
    item.appendChild(create("span", "nav-item-path", op.path));

    item.addEventListener("click", function () {
      selectOperation(op);
      if (window.innerWidth <= 768) {
        el.app.classList.remove("is-sidebar-open");
      }
    });

    return item;
  }

  /* ---------- operation detail ---------- */

  function selectOperation(op) {
    state.operation = op;
    state.values = {};

    // Prefill the body editor with an example payload.
    if (op.requestBody && op.requestBody.schema) {
      var example = window.OpenAPI.exampleFor(op.requestBody.schema, state.model.root);
      state.values.__body = example !== null && example !== undefined
        ? JSON.stringify(example, null, 2)
        : "";
    }

    // Prefill parameters with their declared examples.
    for (var i = 0; i < op.parameters.length; i++) {
      var param = op.parameters[i];
      if (param.example !== undefined && param.example !== null) {
        state.values[param.name] = String(param.example);
      } else if (param.schema && param.schema.default !== undefined) {
        state.values[param.name] = String(param.schema.default);
      } else {
        state.values[param.name] = "";
      }
    }

    renderSidebar();
    renderOperation();
    renderCode();

    el.main.scrollTop = 0;
    updateHash(op);
  }

  function renderOperation() {
    var main = el.mainInner;
    clear(main);

    if (!state.operation) {
      main.appendChild(buildWelcome());
      return;
    }

    var op = state.operation;

    /* header */
    var header = create("div", "op-header");
    var path = create("div", "op-path");
    path.appendChild(
      create("span", "method method-" + op.method.toLowerCase(), op.method)
    );
    path.appendChild(create("span", "op-path-text", op.path));
    header.appendChild(path);

    if (op.summary) {
      header.appendChild(create("h1", "op-summary", op.summary));
    }
    if (op.description) {
      var desc = create("div", "op-description");
      desc.innerHTML = renderMarkdown(op.description);
      header.appendChild(desc);
    }

    var meta = create("div", "op-meta");
    for (var t = 0; t < op.tags.length; t++) {
      meta.appendChild(create("span", "tag", op.tags[t]));
    }
    if (op.deprecated) {
      var dep = create("span", "tag deprecated", "已废弃");
      meta.appendChild(dep);
    }
    if (op.operationId) {
      meta.appendChild(create("span", "tag", op.operationId));
    }
    if (meta.childNodes.length) header.appendChild(meta);
    main.appendChild(header);

    /* parameters */
    if (op.parameters.length) {
      var params = create("section", "section");
      params.appendChild(create("h2", "section-title", "参数"));
      params.appendChild(buildParameterList(op));
      main.appendChild(params);
    }

    /* request body */
    if (op.requestBody) {
      var body = create("section", "section");
      body.appendChild(create("h2", "section-title", "请求体"));
      body.appendChild(buildBodyEditor(op));
      main.appendChild(body);
    }

    /* responses */
    if (op.responses.length) {
      var responses = create("section", "section");
      responses.appendChild(create("h2", "section-title", "响应"));
      responses.appendChild(buildResponseList(op));
      main.appendChild(responses);
    }

    /* try it */
    var tryit = create("section", "section");
    tryit.appendChild(create("h2", "section-title", "发起请求"));
    tryit.appendChild(buildTryIt(op));
    main.appendChild(tryit);
  }

  function buildWelcome() {
    var wrap = create("div", "empty");
    wrap.innerHTML =
      '<svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" ' +
      'stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">' +
      '<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"></path>' +
      '<path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"></path>' +
      "</svg>";
    wrap.appendChild(create("div", "empty-title", "从左侧选择一个接口"));
    wrap.appendChild(
      create(
        "div",
        "empty-hint",
        "接口列表按分组展示，搜索框支持按路径、方法、摘要和标签过滤。"
      )
    );
    return wrap;
  }

  function buildParameterList(op) {
    var list = create("div", "param-list");

    for (var i = 0; i < op.parameters.length; i++) {
      (function (param) {
        var row = create("div", "param");

        var head = create("div", "param-head");
        head.appendChild(create("span", "param-name", param.name));
        head.appendChild(create("span", "param-in", param.in));
        if (param.required) {
          head.appendChild(create("span", "param-required", "必填"));
        }
        if (param.schema) {
          var type = window.OpenAPI.schemaType(param.schema);
          head.appendChild(
            create("span", "param-type", type + formatSuffix(param.schema))
          );
        }
        row.appendChild(head);

        if (param.description) {
          var desc = create("p", "param-desc");
          desc.innerHTML = renderMarkdown(param.description);
          row.appendChild(desc);
        }

        var input = create("input", "param-input");
        input.type = "text";
        input.placeholder = placeholderFor(param);
        input.value = state.values[param.name] || "";
        input.addEventListener("input", function () {
          state.values[param.name] = input.value;
          renderCode();
        });
        row.appendChild(input);

        list.appendChild(row);
      })(op.parameters[i]);
    }

    return list;
  }

  function formatSuffix(schema) {
    if (!schema) return "";
    var constraints = window.OpenAPI.schemaConstraints(schema);
    return constraints.length ? " · " + constraints.join(" · ") : "";
  }

  function placeholderFor(param) {
    if (!param.schema) return "";
    if (param.example !== undefined && param.example !== null) {
      return String(param.example);
    }
    var type = window.OpenAPI.schemaType(param.schema);
    if (type === "integer" || type === "number") return "0";
    if (type === "boolean") return "true";
    if (param.in === "path") return "{" + param.name + "}";
    return "填写 " + param.name;
  }

  function buildBodyEditor(op) {
    var wrap = create("div");

    if (op.requestBody.description) {
      var desc = create("p", "param-desc");
      desc.innerHTML = renderMarkdown(op.requestBody.description);
      wrap.appendChild(desc);
    }

    var editor = create("textarea", "body-editor");
    editor.spellcheck = false;
    editor.value = state.values.__body || "";
    editor.placeholder = '{ "key": "value" }';

    var error = create("p", "body-error");
    error.style.display = "none";

    editor.addEventListener("input", function () {
      state.values.__body = editor.value;
      error.style.display = "none";
      editor.classList.remove("is-invalid");
      renderCode();
    });

    wrap.appendChild(editor);
    wrap.appendChild(error);
    wrap.__editor = editor;
    wrap.__error = error;
    return wrap;
  }

  function buildResponseList(op) {
    var list = create("div", "resp-list");

    for (var i = 0; i < op.responses.length; i++) {
      (function (resp) {
        var item = create("div", "resp");
        if (i > 0) item.classList.add("is-collapsed");

        var head = create("button", "resp-head");
        head.type = "button";

        var codeClass = "resp-code-" + resp.code.charAt(0);
        head.appendChild(create("span", "resp-code " + codeClass, resp.code));
        head.appendChild(create("span", "resp-desc", resp.description));

        var body = create("div", "resp-body");
        if (resp.schema) {
          body.appendChild(buildSchemaTree(resp.schema, state.model.root, 0));
        } else {
          body.appendChild(
            create("div", "param-desc", "该响应没有定义结构。")
          );
        }

        head.addEventListener("click", function () {
          item.classList.toggle("is-collapsed");
        });

        item.appendChild(head);
        item.appendChild(body);
        list.appendChild(item);
      })(op.responses[i]);
    }

    return list;
  }

  /* ---------- schema tree ---------- */

  /* Builds a schema tree for a schema.
   *
   * name is the property name, used for the fields of an object. It is
   * empty for a root schema that has no name of its own.
   */
  function buildSchemaTree(schema, root, depth, name) {
    if (depth > 12) return create("div", "schema-desc", "…");

    var resolved = window.OpenAPI.deref(schema, root);
    if (!resolved) {
      return create("div", "schema-desc", "无法解析的引用");
    }

    var wrapper = create("div", "schema");
    var row = create("div", "schema-row");

    var label = name || schemaNameOf(schema);
    if (label) {
      row.appendChild(create("span", "schema-name", label));
    }

    var type = window.OpenAPI.schemaType(resolved);
    if (type !== label) {
      row.appendChild(create("span", "schema-type", type));
    }
    if (resolved.format) {
      row.appendChild(create("span", "schema-format", resolved.format));
    }

    var constraints = window.OpenAPI.schemaConstraints(resolved);
    for (var c = 0; c < constraints.length; c++) {
      row.appendChild(create("span", "schema-constraint", constraints[c]));
    }

    if (resolved.description) {
      var desc = create("p", "schema-desc");
      desc.innerHTML = renderMarkdown(resolved.description);
      row.appendChild(desc);
    }

    if (resolved.enum && resolved.enum.length) {
      var enumBox = create("div", "schema-enum");
      for (var e = 0; e < resolved.enum.length; e++) {
        enumBox.appendChild(
          create("span", "enum-value", JSON.stringify(resolved.enum[e]))
        );
      }
      row.appendChild(enumBox);
    }

    var body = create("div", "schema-body");
    var hasChildren = false;

    if (resolved.properties) {
      var required = resolved.required || [];
      var keys = Object.keys(resolved.properties);
      for (var p = 0; p < keys.length; p++) {
        var prop = resolved.properties[keys[p]];
        var node = create("div", "schema-node");
        node.appendChild(buildSchemaTree(prop, root, depth + 1, keys[p]));

        if (required.indexOf(keys[p]) !== -1) {
          var propRow = node.querySelector(".schema-row");
          if (propRow) {
            propRow.appendChild(create("span", "schema-required", "必填"));
          }
        }
        body.appendChild(node);
        hasChildren = true;
      }
    }

    if (resolved.items) {
      var itemNode = create("div", "schema-node");
      itemNode.appendChild(buildSchemaTree(resolved.items, root, depth + 1, "items"));
      body.appendChild(itemNode);
      hasChildren = true;
    }

    if (hasChildren) {
      var toggle = create("button", "schema-toggle", "收起");
      toggle.type = "button";
      toggle.addEventListener("click", function (event) {
        event.stopPropagation();
        var collapsed = wrapper.classList.toggle("is-collapsed");
        toggle.textContent = collapsed ? "展开" : "收起";
      });
      row.appendChild(toggle);
    }

    wrapper.appendChild(row);
    wrapper.appendChild(body);
    return wrapper;
  }

  function schemaNameOf(schema) {
    if (schema && typeof schema.$ref === "string") {
      return schema.$ref.split("/").pop();
    }
    return "";
  }

  /* ---------- code panel ---------- */

  function renderCode() {
    var tabs = el.codeTabs;
    var body = el.codeBody;
    clear(tabs);

    if (!state.operation) {
      body.textContent = "";
      return;
    }

    var languages = window.CodeSamples.languages;
    for (var i = 0; i < languages.length; i++) {
      (function (language) {
        var tab = create("button", "code-tab", language.name);
        tab.type = "button";
        if (language.id === state.language) tab.classList.add("is-active");
        tab.addEventListener("click", function () {
          state.language = language.id;
          renderCode();
        });
        tabs.appendChild(tab);
      })(languages[i]);
    }

    var actions = create("div", "code-actions");
    var copy = create("button", "btn btn-ghost btn-sm", "复制");
    copy.type = "button";
    copy.addEventListener("click", function () {
      var text = body.textContent;
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(
          function () {
            showToast("已复制到剪贴板");
          },
          function () {
            showToast("复制失败，请手动选择");
          }
        );
      } else {
        showToast("当前浏览器不支持自动复制");
      }
    });
    actions.appendChild(copy);
    tabs.appendChild(actions);

    var snippet = window.CodeSamples.generate(
      state.language,
      state.operation,
      state.values
    );
    body.innerHTML = highlight(snippet, state.language);
  }

  /* ---------- try it ---------- */

  function buildTryIt(op) {
    var wrap = create("div", "tryit");

    var row = create("div", "tryit-row");
    row.appendChild(create("span", "tryit-label", "地址"));

    var url = create("input", "tryit-input");
    url.type = "text";
    url.readOnly = true;
    url.value = resolveUrl(op);
    row.appendChild(url);
    wrap.appendChild(row);

    /* auth */
    var schemes = state.model.securitySchemes || {};
    var names = Object.keys(schemes);
    if (names.length) {
      var authRow = create("div", "tryit-row");
      authRow.appendChild(create("span", "tryit-label", "认证"));
      for (var i = 0; i < names.length; i++) {
        (function (name, scheme) {
          var field = create("input", "tryit-input");
          field.type = "text";
          field.placeholder = authPlaceholder(name, scheme);
          field.dataset.scheme = name;
          field.addEventListener("input", function () {
            state.values["__auth:" + name] = field.value;
          });
          authRow.appendChild(field);
        })(names[i], schemes[names[i]]);
      }
      wrap.appendChild(authRow);
    }

    var buttonRow = create("div", "tryit-row");
    var send = create("button", "btn", "发送请求");
    send.type = "button";
    send.addEventListener("click", function () {
      executeRequest(op, send, resultBox);
    });
    buttonRow.appendChild(send);
    wrap.appendChild(buttonRow);

    var resultBox = create("div");
    wrap.appendChild(resultBox);

    return wrap;
  }

  /* Builds the placeholder of an authentication field. */
  function authPlaceholder(name, scheme) {
    if (scheme.type === "http") {
      return "Bearer token（" + name + "）";
    }
    if (scheme.type === "apiKey") {
      return (scheme.name || "API key") + "（" + name + "）";
    }
    if (scheme.type === "oauth2") {
      return "访问令牌（" + name + "）";
    }
    return name;
  }

  /* Builds the absolute URL of an operation against the spec server. */
  function resolveUrl(op) {
    var servers = state.model.servers || [];
    var base = "";
    if (servers.length) {
      base = servers[0].url || "";
    }
    if (!base) base = window.location.origin;

    var request = window.CodeSamples.buildRequest(op, state.values);
    return base.replace(/\/$/, "") + request.url;
  }

  function executeRequest(op, button, resultBox) {
    var request = window.CodeSamples.buildRequest(op, state.values);

    var headers = {};
    for (var name in request.headers) {
      if (Object.prototype.hasOwnProperty.call(request.headers, name)) {
        headers[name] = request.headers[name];
      }
    }

    /* Apply the credentials the user typed in the auth fields. */
    var schemes = state.model.securitySchemes || {};
    for (var key in state.values) {
      if (key.indexOf("__auth:") !== 0) continue;
      var schemeName = key.slice(7);
      var value = state.values[key];
      if (!value) continue;
      var scheme = schemes[schemeName];
      if (!scheme) continue;
      if (scheme.type === "http" && scheme.scheme === "bearer") {
        headers["Authorization"] = "Bearer " + value;
      } else if (scheme.type === "apiKey" && scheme.in === "header") {
        headers[scheme.name || "Authorization"] = value;
      }
    }

    if (request.body) {
      headers["Content-Type"] = request.contentType;
    }

    var init = {
      method: request.method,
      headers: headers,
    };
    if (request.body) init.body = request.body;

    var url = resolveUrl(op);

    button.disabled = true;
    button.textContent = "请求中…";
    clear(resultBox);

    var started = Date.now();
    fetch(url, init)
      .then(function (response) {
        return response.text().then(function (text) {
          var elapsed = Date.now() - started;
          showResult(resultBox, response.status, response.statusText, text, elapsed);
        });
      })
      .catch(function (error) {
        var elapsed = Date.now() - started;
        showResult(
          resultBox,
          0,
          "请求失败",
          String((error && error.message) || error),
          elapsed
        );
      })
      .then(function () {
        button.disabled = false;
        button.textContent = "发送请求";
      });
  }

  function showResult(box, status, statusText, text, elapsed) {
    clear(box);

    var result = create("div", "result");
    var head = create("div", "result-head");

    var statusEl = create("span", "result-status", status ? String(status) : "ERR");
    if (status >= 200 && status < 300) statusEl.classList.add("is-ok");
    else if (status >= 300 && status < 400) statusEl.classList.add("is-redirect");
    else if (status > 0) statusEl.classList.add("is-error");
    else statusEl.classList.add("is-error");
    head.appendChild(statusEl);

    head.appendChild(create("span", "resp-desc", statusText || ""));
    head.appendChild(create("span", "result-time", elapsed + " ms"));
    result.appendChild(head);

    var body = create("pre", "result-body");
    var pretty = text;
    try {
      pretty = JSON.stringify(JSON.parse(text), null, 2);
    } catch (e) {
      /* keep the raw text */
    }
    body.textContent = pretty;
    result.appendChild(body);

    box.appendChild(result);
  }

  /* ---------- routing ---------- */

  function updateHash(op) {
    if (!window.history || !window.history.replaceState) return;
    var hash = "#" + encodeURIComponent(op.method.toLowerCase() + " " + op.path);
    window.history.replaceState(null, "", hash);
  }

  function operationFromHash() {
    if (!window.location.hash) return null;
    var text = decodeURIComponent(window.location.hash.slice(1));
    for (var i = 0; i < state.model.operations.length; i++) {
      var op = state.model.operations[i];
      if (op.method.toLowerCase() + " " + op.path === text) return op;
    }
    return null;
  }

  /* ---------- theme ---------- */

  function applyTheme(theme) {
    document.documentElement.setAttribute("data-theme", theme);
    try {
      window.localStorage.setItem("openapidoc-theme", theme);
    } catch (e) {
      /* storage may be unavailable */
    }
  }

  function initTheme() {
    var stored = null;
    try {
      stored = window.localStorage.getItem("openapidoc-theme");
    } catch (e) {
      /* ignore */
    }
    if (!stored) {
      stored =
        window.matchMedia &&
        window.matchMedia("(prefers-color-scheme: dark)").matches
          ? "dark"
          : "light";
    }
    applyTheme(stored);
  }

  function toggleTheme() {
    var current = document.documentElement.getAttribute("data-theme");
    applyTheme(current === "dark" ? "light" : "dark");
  }

  /* ---------- boot ---------- */

  function bind() {
    el = {
      app: $("#app"),
      nav: $("#nav"),
      main: $("#main"),
      mainInner: $("#main-inner"),
      search: $("#search"),
      codeTabs: $("#code-tabs"),
      codeBody: $("#code-body"),
      toast: $("#toast"),
      themeBtn: $("#theme-toggle"),
      menuBtn: $("#menu-toggle"),
      backdrop: $("#sidebar-backdrop"),
    };

    el.search.addEventListener("input", function () {
      state.query = el.search.value;
      renderSidebar();
    });

    el.themeBtn.addEventListener("click", toggleTheme);
    el.menuBtn.addEventListener("click", function () {
      el.app.classList.toggle("is-sidebar-open");
    });
    el.backdrop.addEventListener("click", function () {
      el.app.classList.remove("is-sidebar-open");
    });

    document.addEventListener("keydown", function (event) {
      if (event.key === "/" && document.activeElement !== el.search) {
        event.preventDefault();
        el.search.focus();
        el.search.select();
      }
      if (event.key === "Escape" && document.activeElement === el.search) {
        el.search.value = "";
        state.query = "";
        renderSidebar();
        el.search.blur();
      }
    });

    window.addEventListener("hashchange", function () {
      var op = operationFromHash();
      if (op && (!state.operation || op.id !== state.operation.id)) {
        selectOperation(op);
      }
    });
  }

  function load() {
    el.mainInner.innerHTML =
      '<div class="skeleton" style="width:60%"></div>' +
      '<div class="skeleton" style="width:90%"></div>' +
      '<div class="skeleton" style="width:75%"></div>';

    fetch(state.specUrl, { headers: { Accept: "application/json" } })
      .then(function (response) {
        if (!response.ok) {
          throw new Error("HTTP " + response.status);
        }
        return response.json();
      })
      .then(function (doc) {
        state.model = window.OpenAPI.parse(doc);
        state.groups = state.model.groups;

        document.title = (state.model.info.title || "API") + " · " + state.title;
        var titleEl = $("#doc-title");
        if (titleEl) titleEl.textContent = state.model.info.title || state.title;
        var versionEl = $("#doc-version");
        if (versionEl) versionEl.textContent = state.model.info.version || "";
        var descEl = $("#doc-desc");
        if (descEl && state.model.info.description) {
          descEl.textContent = state.model.info.description;
        }

        renderSidebar();

        var initial = operationFromHash() || state.model.operations[0];
        if (initial) {
          selectOperation(initial);
        } else {
          renderOperation();
        }
      })
      .catch(function (error) {
        el.mainInner.innerHTML = "";
        var empty = create("div", "empty");
        empty.appendChild(create("div", "empty-title", "无法加载 OpenAPI 文档"));
        empty.appendChild(
          create("div", "empty-hint", String((error && error.message) || error))
        );
        empty.appendChild(
          create("div", "empty-hint", "地址：" + state.specUrl)
        );
        el.mainInner.appendChild(empty);
      });
  }

  function start() {
    var config = window.__OPENAPIDOC__ || {};
    state.specUrl = config.specUrl || "/openapi.json";
    state.title = config.title || "API Documentation";

    initTheme();
    bind();
    load();

    // The code panel is part of the layout, show it on wide screens.
    if (window.innerWidth > 1024) {
      el.app.classList.add("is-code-open");
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", start);
  } else {
    start();
  }
})();
