import 'elements-sk/styles/buttons'
import 'elements-sk/error-toast-sk'

import { errorMessage } from 'elements-sk/errorMessage'

import { $$ } from 'common-sk/modules/dom'
import { diffDate } from 'common-sk/modules/human'
import { jsonOrThrow } from 'common-sk/modules/jsonOrThrow'

import { html, render } from 'lit-html'

const PathKitInit = require('pathkit-wasm/bin/pathkit.js');

// Main template for this element
const template = (ele) => html`
<header>
  <div class=title>PathKit Fiddle</div>
  <div class=flex></div>
  <div class=version>PathKit Version: <a href="https://www.npmjs.com/package/pathkit-wasm">0.4.0</a></div>
</header>

<main>
  ${codeEditor(ele)}
  <div class=output>
    <div class=buttons>
      <button class=action @click=${() => ele._run()}>Run</button>
      <button class=action @click=${() => ele._save()}>Save</button>
    </div>
    <canvas id=canvas width=500 height=500></canvas>
  </div>
</main>
<footer>
  <error-toast-sk></error-toast-sk>
</footer>`;


const codeEditor = (ele) => html`
<div id=editor>
  <textarea class=code spellcheck=false rows=${lines(ele.content)} cols=80
        @paste=${() => ele._changed()} @input=${() => ele._changed()}
  ></textarea>
  <div class=numbers>
    ${repeat(lines(ele.content)).map((_, n) => _lineNumber(n+1))}
  </div>

</editor>
`

// Returns the number of lines in str, with a minimum of 10
// (because the editor with less than 10 lines looks a bit strange).
function lines(str) {
  // see https://stackoverflow.com/a/4009768
  return Math.max(10, (str.match(/\n/g) || []).length + 1);
}

// repeat returns an array of n 'undefined' which allows
// for repeating a template a fixed number of times
// using map.
function repeat(n) {
  // See https://stackoverflow.com/a/10050831
  return [...Array(n)];
}

const _lineNumber = (n) => html`
  <div id=${'L'+n}>${n}</div>
`;

function resetCanvas(canvas) {
  // Reset the transform of the canvas then re-draw it blank.
  let ctx = canvas.getContext('2d');
  ctx.setTransform(1, 0, 0, 1, 0, 0);
  ctx.clearRect(0, 0, canvas.width, canvas.height);
}

/**
 * @module jsfiddle/modules/pathkit-fiddle
 * @description <h2><code>pathkit-fiddle</code></h2>
 *
 * <p>
 *   The top level element for displaying pathkit fiddles.
 *   The main elements are a code editor box (textarea), a canvas
 *   on which to render the result and a few buttons.
 * </p>
 *
 */
window.customElements.define('pathkit-fiddle', class extends HTMLElement {

  constructor() {
    super();

    this._content = '';
    this.PathKit = null;
    this._editor = null; // set in render to be the textarea
  }

  /** @prop {String} content - The current code in the editor.*/
  get content() { return this._content; }
  set content(c) {
    this._content = c;
    this._render();
    this._editor.value = c;
  }

  connectedCallback() {
    this._render();
    PathKitInit({
      locateFile: (file) => '/res/'+file,
    }).then((PathKit) => {
      this.PathKit = PathKit;
      if (this.content) {
        this._run(); // auto-run the code if the code was loaded.
      }
    });

    if (!this.content) {
      this._loadCode();
    }
    // Listen for the forward and back buttons and re-load the code
    // on any changes. Without this, the url changes, but nothing
    // happens in the DOM.
    window.addEventListener('popstate', this._loadCode.bind(this));
  }

  disconnectedCallback() {
    window.removeEventListener('popstate', this._loadCode.bind(this));
  }

  _changed() {
    this.content = this._editor.value;
  }

  _loadCode() {
    // The location should be either /pathkit or /pathkit/<fiddlehash>
    let path = window.location.pathname;
    let hash = '';
    if (path.length > 9) { // 9 characters in '/pathkit/'
      hash = path.slice(9);
    }

    fetch(`/_/code?type=pathkit&hash=${hash}`)
      .then(jsonOrThrow)
      .then((json) => {
        this.content = json.code;
        if (this.PathKit) {
          this._run(); // auto-run the code if PathKit is loaded.
        }
      }
    ).catch((e) => {
      errorMessage('Fiddle not Found', 10000);
      this.content = '';
      const canvas = $$('#canvas', this);
      resetCanvas(canvas);
    });
  }

  _render() {
    render(template(this), this);
    this._editor = $$('#editor textarea', this);
  }

  _run() {
    if (!this.PathKit) {
      errorMessage('PathKit is still loading.');
      return;
    }

    const canvas = $$('#canvas', this);
    resetCanvas(canvas);
    try {
      let f = new Function('PathKit', 'canvas', // actual params
          // shadow these globals to at least make exploitation harder. CSP
          // is our first line of defense, this adds another layer.
          'window', 'document', 'open', 'event', 'Function', 'eval', 'frames',
          'frameElement', 'localStorage', 'history', 'messageManager', 'name',
          'opener', 'pkcs11', 'self', 'status', 'top', 'visualViewport',
          'caches', 'origin', 'indexedDB', 'Worker', 'openDialog', 'alert',
          'prompt', 'parent',
           this.content); // user given code
      f = f.bind({}); // By default, f is bound to Window.  Re bind it to remove that access.
      f(this.PathKit, canvas);
    } catch(e) {
      errorMessage(e);
    }
  }

  _save() {
    fetch('/_/save', {
      method: 'PUT',
      headers: new Headers({
        'content-type': 'application/json',
      }),
      body: JSON.stringify({
        code: this.content,
        type: 'pathkit',
      })
    }).then(jsonOrThrow).then((json) => {
        history.pushState(null, '', json.new_url);
      }
    ).catch(errorMessage);
  }

});
