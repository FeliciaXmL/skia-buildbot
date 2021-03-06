import { escapeAndLinkify } from './linkify';

describe('formatAndLinkify', function() {
  it('wraps links in anchor tags', function() {
    test('http://example.com',
        '<a href="http://example.com" target="_blank" rel="noopener">http://example.com</a>');
    test('visit https://example.com for more',
         'visit <a href="https://example.com" target="_blank" rel="noopener">'+
         'https://example.com</a> for more');
  });

  it('replaces newlines with <br/>', function() {
    test('hello \n world \r\n good \t to \\ see \r you',
         'hello <br> world <br> good \t to \\ see <br> you');
  });

  it('replaces expands bug shorthand with anchor tags', function() {
    test('skia:1234',
         '<a href="http://skbug.com/1234" target="_blank" rel="noopener">skia:1234</a>');
    test('chromium:123456789101112',
         '<a href="http://crbug.com/123456789101112" target="_blank" rel="noopener">'+
         'chromium:123456789101112</a>');
    test('See skia:1234 and skia:456',
      'See <a href="http://skbug.com/1234" target="_blank" rel="noopener">skia:1234</a> '+
      'and <a href="http://skbug.com/456" target="_blank" rel="noopener">skia:456</a>');
  });

  it('does not replace invalid shorthand', function() {
    test('skia:abcd', 'skia:abcd');
    test('unknown:123', 'unknown:123');
    test('skia:', 'skia:');
  });

  it('escapes tags, to avoid XSS', function() {
    test('<div>foo</div>', '&lt;div&gt;foo&lt;/div&gt;');
    test('<script>alert("xss")</script>', '&lt;script&gt;alert("xss")&lt;/script&gt;');
  });

  it('does all of the above at once', function() {
    test('<b>Hey<b>, check out skia:9000 \n and http://www.example.com/cool/thing?special=true',
      '&lt;b&gt;Hey&lt;b&gt;, check out '+
      '<a href="http://skbug.com/9000" target="_blank" rel="noopener">skia:9000</a> '+
      '<br> and <a href="http://www.example.com/cool/thing?special=true" '+
      'target="_blank" rel="noopener">http://www.example.com/cool/thing?special=true</a>');
  })
});

const test = (input, output) => {
  expect(escapeAndLinkify(input).innerHTML).to.equal(output);
};
