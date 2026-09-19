/* ==========================================================================
   cnc-dialog.js — build an ncurses-style modal form from a field spec.

   Usage:

     const values = await cncDialog({
       title: "Job Parameters",
       blurb: "Optional paragraph above the fields.",
       fields: [
         { name: "jobname", label: "Job name", type: "text", value: "untitled" },
         { name: "width",   label: "Width",    type: "number", value: 100,
           step: 0.001, min: 0, unit: "mm" },
         { name: "side",    label: "Cutter offset", type: "radio",
           value: "inside",
           options: [
             { value: "inside",  label: "Inside — cut within the boundary" },
             { value: "outside", label: "Outside — cut around the boundary" }
           ]},
         { name: "climb",   label: "Climb milling", type: "check", value: true },
         { name: "units",   label: "Units", type: "select", value: "mm",
           options: [{ value: "mm", label: "Millimetres" },
                     { value: "in", label: "Inches" }] }
       ]
     });

   Resolves to an object of name -> value, or null if cancelled
   (Cancel button or Esc). Numbers come back as numbers, checks as booleans.

   Field types: text, number, radio, check, select.
   Per-field extras: unit, min, max, step, maxlength, required (default true
   for text/number), placeholder, hint.
   Dialog extras: okLabel, cancelLabel, hotOk, hotCancel, width, labelWidth.
   ========================================================================== */

(function (global) {
  "use strict";

  const esc = (s) => String(s).replace(/[&<>"]/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));

  // "< OK >" with the first matching letter picked out in red
  function buttonLabel(text, hot) {
    const i = hot ? text.toUpperCase().indexOf(hot.toUpperCase()) : 0;
    if (i < 0) return "&lt; &nbsp;" + esc(text) + "&nbsp; &gt;";
    return "&lt; &nbsp;" + esc(text.slice(0, i)) +
           '<span class="cd-hot">' + esc(text[i]) + "</span>" +
           esc(text.slice(i + 1)) + "&nbsp; &gt;";
  }

  function labelWidth(fields) {
    const longest = fields.reduce(
      (n, f) => Math.max(n, (f.label || "").length), 0);
    return Math.min(Math.max(longest + 2, 10), 24) + "ch";
  }

  function renderField(f, uid) {
    const id = uid + "-" + f.name;
    const req = f.required === undefined
      ? (f.type === "text" || f.type === "number" || f.type === undefined)
      : f.required;
    const attrs = [
      'id="' + id + '"',
      'name="' + esc(f.name) + '"',
      req ? "required" : "",
      f.min !== undefined ? 'min="' + f.min + '"' : "",
      f.max !== undefined ? 'max="' + f.max + '"' : "",
      f.step !== undefined ? 'step="' + f.step + '"' : "",
      f.maxlength ? 'maxlength="' + f.maxlength + '"' : "",
      f.placeholder ? 'placeholder="' + esc(f.placeholder) + '"' : ""
    ].filter(Boolean).join(" ");

    const unit = f.unit ? '<span class="cd-unit">' + esc(f.unit) + "</span>" : "";
    const hint = f.hint ? '<span class="cd-unit">' + esc(f.hint) + "</span>" : "";

    switch (f.type) {
      case "radio":
      case "check": {
        const isCheck = f.type === "check";
        const cls = "cd-choice" + (isCheck ? " cd-check" : "");
        if (isCheck) {
          return '<label class="' + cls + '">' +
            '<input type="checkbox" id="' + id + '" name="' + esc(f.name) + '"' +
            (f.value ? " checked" : "") + ">" +
            '<span class="cd-mark"></span>' +
            '<span class="cd-text">' + esc(f.label) + "</span></label>";
        }
        const opts = (f.options || []).map((o) =>
          '<label class="cd-choice">' +
          '<input type="radio" name="' + esc(f.name) + '" value="' + esc(o.value) + '"' +
          (o.value === f.value ? " checked" : "") + ">" +
          '<span class="cd-mark"></span>' +
          '<span class="cd-text">' + esc(o.label) + "</span></label>").join("");
        return "<fieldset><legend>" + esc(f.label) + "</legend>" + opts + "</fieldset>";
      }

      case "select": {
        const opts = (f.options || []).map((o) =>
          '<option value="' + esc(o.value) + '"' +
          (o.value === f.value ? " selected" : "") + ">" +
          esc(o.label) + "</option>").join("");
        return '<div class="cd-row"><label for="' + id + '">' + esc(f.label) +
          '</label><span class="cd-input"><select ' + attrs + ">" + opts +
          "</select>" + hint + "</span></div>";
      }

      case "number":
        return '<div class="cd-row"><label for="' + id + '">' + esc(f.label) +
          '</label><span class="cd-input"><input type="number" ' + attrs +
          ' value="' + esc(f.value === undefined ? "" : f.value) + '">' +
          unit + hint + "</span></div>";

      default:
        return '<div class="cd-row"><label for="' + id + '">' + esc(f.label) +
          '</label><span class="cd-input"><input type="text" ' + attrs +
          ' value="' + esc(f.value === undefined ? "" : f.value) + '">' +
          unit + hint + "</span></div>";
    }
  }

  function cncDialog(spec) {
    const fields = spec.fields || [];
    const uid = "cd" + Math.random().toString(36).slice(2, 8);

    const dlg = document.createElement("dialog");
    dlg.className = "cd";
    dlg.style.setProperty("--cd-labelw", spec.labelWidth || labelWidth(fields));
    if (spec.width) dlg.style.width = spec.width;

    dlg.innerHTML =
      '<h2 class="cd-title"><span>&#9508; ' + esc(spec.title || "") +
        " &#9500;</span></h2>" +
      (spec.blurb ? '<p class="cd-blurb">' + esc(spec.blurb) + "</p>" : "") +
      '<form novalidate="false">' +
        fields.map((f) => renderField(f, uid)).join("") +
        '<div class="cd-buttons">' +
          '<button class="cd-btn" type="submit">' +
            buttonLabel(spec.okLabel || "OK", spec.hotOk || "O") + "</button>" +
          '<button class="cd-btn" type="button" data-cancel>' +
            buttonLabel(spec.cancelLabel || "Cancel", spec.hotCancel || "C") +
          "</button>" +
        "</div>" +
      "</form>";

    document.body.appendChild(dlg);

    return new Promise((resolve) => {
      let settled = false;
      const done = (val) => {
        if (settled) return;
        settled = true;
        dlg.close();
        dlg.remove();
        resolve(val);
      };

      const form = dlg.querySelector("form");

      form.addEventListener("submit", (e) => {
        e.preventDefault();
        if (!form.reportValidity()) return;
        const raw = Object.fromEntries(new FormData(form).entries());
        const out = {};
        fields.forEach((f) => {
          if (f.type === "check") out[f.name] = form.elements[f.name].checked;
          else if (f.type === "number") out[f.name] = parseFloat(raw[f.name]);
          else out[f.name] = raw[f.name];
        });
        done(out);
      });

      dlg.querySelector("[data-cancel]").addEventListener("click", () => done(null));
      dlg.addEventListener("cancel", (e) => { e.preventDefault(); done(null); });

      dlg.showModal();
      const first = dlg.querySelector("input, select");
      if (first) { first.focus(); if (first.select) first.select(); }
    });
  }

  global.cncDialog = cncDialog;
})(window);
