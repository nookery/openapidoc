/* openapidoc native UI - OpenAPI parsing layer
 *
 * Turns an OpenAPI 3.x document into a flat, navigable model. Everything
 * here is pure data transformation, no DOM access.
 */
(function (global) {
  "use strict";

  var HTTP_METHODS = [
    "get", "put", "post", "delete", "options", "head", "patch", "trace",
  ];

  /* ---------- helpers ---------- */

  function isObject(value) {
    return value !== null && typeof value === "object" && !Array.isArray(value);
  }

  /* Resolves a local JSON pointer such as "#/components/schemas/User". */
  function resolveRef(ref, root) {
    if (typeof ref !== "string" || ref.charAt(0) !== "#") return null;
    var path = ref.slice(1).split("/");
    var current = root;
    for (var i = 0; i < path.length; i++) {
      var segment = path[i].replace(/~1/g, "/").replace(/~0/g, "~");
      // The pointer starts with a slash, so the first segment is empty.
      if (segment === "") continue;
      if (current === null || current === undefined) return null;
      current = current[segment];
    }
    return current === undefined ? null : current;
  }

  /* Follows $ref chains until a concrete object is reached. */
  function deref(node, root, seen) {
    if (!isObject(node)) return node;
    if (typeof node.$ref !== "string") return node;
    seen = seen || {};
    if (seen[node.$ref]) return null;
    seen[node.$ref] = true;
    var target = resolveRef(node.$ref, root);
    if (!target) return null;
    return deref(target, root, seen);
  }

  /* ---------- schema helpers ---------- */

  /* Returns the effective type of a schema as a readable string. */
  function schemaType(schema) {
    if (!isObject(schema)) return "any";
    if (schema.type) {
      if (Array.isArray(schema.type)) return schema.type.join(" | ");
      return schema.type;
    }
    if (schema.properties) return "object";
    if (schema.items) return "array";
    if (schema.enum) return "enum";
    if (schema.$ref) {
      var name = schema.$ref.split("/").pop();
      return name;
    }
    return "any";
  }

  /* Collects human readable constraints of a schema. */
  function schemaConstraints(schema) {
    if (!isObject(schema)) return [];
    var out = [];
    if (schema.minimum !== undefined) out.push("≥ " + schema.minimum);
    if (schema.maximum !== undefined) out.push("≤ " + schema.maximum);
    if (schema.exclusiveMinimum !== undefined) {
      out.push("> " + schema.exclusiveMinimum);
    }
    if (schema.exclusiveMaximum !== undefined) {
      out.push("< " + schema.exclusiveMaximum);
    }
    if (schema.minLength !== undefined) out.push("minLength " + schema.minLength);
    if (schema.maxLength !== undefined) out.push("maxLength " + schema.maxLength);
    if (schema.pattern) out.push("pattern " + schema.pattern);
    if (schema.minItems !== undefined) out.push("minItems " + schema.minItems);
    if (schema.maxItems !== undefined) out.push("maxItems " + schema.maxItems);
    if (schema.uniqueItems) out.push("unique");
    return out;
  }

  /* Produces an example value for a schema, used to prefill the body editor. */
  function exampleFor(schema, root, depth) {
    depth = depth || 0;
    if (!isObject(schema) || depth > 6) return null;

    var resolved = deref(schema, root);
    if (!isObject(resolved)) return null;

    if (resolved.example !== undefined) return resolved.example;
    if (resolved.default !== undefined) return resolved.default;

    if (resolved.enum && resolved.enum.length) return resolved.enum[0];

    var type = schemaType(resolved);
    if (type === "object" || resolved.properties) {
      var out = {};
      var props = resolved.properties || {};
      for (var key in props) {
        if (!Object.prototype.hasOwnProperty.call(props, key)) continue;
        var value = exampleFor(props[key], root, depth + 1);
        if (value !== null && value !== undefined) out[key] = value;
      }
      return out;
    }
    if (type === "array" || resolved.items) {
      var item = exampleFor(resolved.items, root, depth + 1);
      return item === null || item === undefined ? [] : [item];
    }
    switch (type) {
      case "string":
        return resolved.format === "date-time"
          ? "2024-01-01T00:00:00Z"
          : resolved.format === "date"
            ? "2024-01-01"
            : "string";
      case "integer":
      case "number":
        return 0;
      case "boolean":
        return true;
      case "null":
        return null;
      default:
        return null;
    }
  }

  /* ---------- operation model ---------- */

  /* Builds the flat list of operations from the paths object. */
  function collectOperations(doc) {
    var operations = [];
    var paths = doc.paths || {};
    var root = doc;

    for (var path in paths) {
      if (!Object.prototype.hasOwnProperty.call(paths, path)) continue;
      var item = paths[path];
      if (!isObject(item)) continue;

      var sharedParams = Array.isArray(item.parameters) ? item.parameters : [];

      for (var i = 0; i < HTTP_METHODS.length; i++) {
        var method = HTTP_METHODS[i];
        var op = item[method];
        if (!isObject(op)) continue;

        var parameters = sharedParams.slice();
        if (Array.isArray(op.parameters)) {
          parameters = parameters.concat(op.parameters);
        }

        operations.push({
          id: method + " " + path,
          method: method.toUpperCase(),
          path: path,
          summary: op.summary || "",
          description: op.description || "",
          operationId: op.operationId || "",
          tags: Array.isArray(op.tags) ? op.tags : [],
          deprecated: op.deprecated === true,
          parameters: parameters.map(function (p) {
            return normaliseParameter(p, root);
          }),
          requestBody: normaliseRequestBody(op.requestBody, root),
          responses: normaliseResponses(op.responses, root),
          security: Array.isArray(op.security) ? op.security : null,
        });
      }
    }

    operations.sort(function (a, b) {
      if (a.path === b.path) {
        return HTTP_METHODS.indexOf(a.method.toLowerCase()) -
          HTTP_METHODS.indexOf(b.method.toLowerCase());
      }
      return a.path < b.path ? -1 : 1;
    });

    return operations;
  }

  /* Normalises a single parameter object. */
  function normaliseParameter(param, root) {
    if (!isObject(param)) {
      return { name: "", in: "", required: false, description: "", schema: null };
    }
    var resolved = deref(param, root) || param;
    var schema = resolved.schema ? deref(resolved.schema, root) : null;
    return {
      name: resolved.name || "",
      in: resolved.in || "query",
      description: resolved.description || "",
      required: resolved.required === true,
      deprecated: resolved.deprecated === true,
      schema: schema,
      example: resolved.example !== undefined
        ? resolved.example
        : schema && schema.example !== undefined
          ? schema.example
          : undefined,
    };
  }

  /* Normalises the request body object. */
  function normaliseRequestBody(body, root) {
    if (!isObject(body)) return null;
    var resolved = deref(body, root) || body;
    var content = resolved.content || {};
    var contentType = "application/json";
    var media = content[contentType];
    if (!media) {
      var keys = Object.keys(content);
      if (keys.length) {
        contentType = keys[0];
        media = content[contentType];
      }
    }
    if (!isObject(media)) return null;
    return {
      description: resolved.description || "",
      required: resolved.required === true,
      contentType: contentType,
      schema: media.schema ? deref(media.schema, root) : null,
    };
  }

  /* Normalises the responses object. */
  function normaliseResponses(responses, root) {
    var out = [];
    if (!isObject(responses)) return out;
    for (var code in responses) {
      if (!Object.prototype.hasOwnProperty.call(responses, code)) continue;
      var resp = deref(responses[code], root);
      if (!isObject(resp)) continue;
      var content = resp.content || {};
      var contentType = null;
      var media = null;
      var keys = Object.keys(content);
      if (keys.length) {
        contentType = keys[0];
        media = content[contentType];
      }
      out.push({
        code: code,
        description: resp.description || "",
        contentType: contentType,
        schema: media && media.schema ? deref(media.schema, root) : null,
      });
    }
    out.sort(function (a, b) {
      if (a.code === "default") return 1;
      if (b.code === "default") return -1;
      return Number(a.code) - Number(b.code);
    });
    return out;
  }

  /* ---------- grouping ---------- */

  /* Groups operations by their first tag. */
  function groupOperations(operations) {
    var groups = [];
    var index = {};

    for (var i = 0; i < operations.length; i++) {
      var op = operations[i];
      var name = op.tags && op.tags.length ? op.tags[0] : "Endpoints";
      if (!index[name]) {
        index[name] = { name: name, operations: [] };
        groups.push(index[name]);
      }
      index[name].operations.push(op);
    }

    groups.sort(function (a, b) {
      if (a.name === "Endpoints") return 1;
      if (b.name === "Endpoints") return -1;
      return a.name < b.name ? -1 : 1;
    });

    return groups;
  }

  /* ---------- public API ---------- */

  global.OpenAPI = {
    HTTP_METHODS: HTTP_METHODS,

    /* Parses a document into the model the UI renders. */
    parse: function (doc) {
      var operations = collectOperations(doc);
      return {
        info: doc.info || {},
        servers: doc.servers || [],
        securitySchemes: (doc.components && doc.components.securitySchemes) || {},
        operations: operations,
        groups: groupOperations(operations),
        schemas: (doc.components && doc.components.schemas) || {},
        root: doc,
      };
    },

    /* Exposed helpers used by the render layer. */
    deref: deref,
    resolveRef: resolveRef,
    schemaType: schemaType,
    schemaConstraints: schemaConstraints,
    exampleFor: exampleFor,
    isObject: isObject,
  };
})(window);
