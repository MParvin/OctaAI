/**
 * Content Script - DOM manipulation and data extraction
 * Handles all 12 command types for browser automation
 */

// Keep track of command execution
let commandQueue = [];
let currentCommand = null;

/**
 * Handle messages from background script
 */
browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'execute_command') {
    executeCommand(message.command)
      .then(result => {
        sendResponse({
          success: true,
          result
        });
      })
      .catch(error => {
        sendResponse({
          success: false,
          error: error.message
        });
      });
    return true; // Indicate we'll respond asynchronously
  }
});

/**
 * Execute a command from the daemon
 */
async function executeCommand(command) {
  console.log('[Content] Executing command:', command.type, command.params);
  currentCommand = command;

  try {
    let result;
    switch (command.type) {
      case 'navigate':
        result = await cmdNavigate(command.params);
        break;
      case 'click':
        result = await cmdClick(command.params);
        break;
      case 'fill':
        result = await cmdFill(command.params);
        break;
      case 'submit':
        result = await cmdSubmit(command.params);
        break;
      case 'extract':
        result = await cmdExtract(command.params);
        break;
      case 'screenshot':
        result = await cmdScreenshot(command.params);
        break;
      case 'execute':
        result = await cmdExecute(command.params);
        break;
      case 'wait_for':
        result = await cmdWaitFor(command.params);
        break;
      case 'scroll':
        result = await cmdScroll(command.params);
        break;
      case 'get_cookies':
        result = await cmdGetCookies(command.params);
        break;
      case 'set_cookies':
        result = await cmdSetCookies(command.params);
        break;
      case 'get_page_source':
        result = await cmdGetPageSource(command.params);
        break;
      default:
        throw new Error(`Unknown command type: ${command.type}`);
    }

    return {
      status: 'success',
      result,
      pageState: getPageState()
    };
  } catch (error) {
    console.error('[Content] Command failed:', error);
    return {
      status: 'error',
      error: error.message,
      pageState: getPageState()
    };
  }
}

/**
 * 1. Navigate to a URL
 */
async function cmdNavigate(params) {
  if (!params.url) {
    throw new Error('URL is required for navigate command');
  }
  window.location.href = params.url;
  // Wait for navigation to complete
  await new Promise(resolve => setTimeout(resolve, 2000));
  return { success: true, url: window.location.href };
}

/**
 * 2. Click an element
 */
async function cmdClick(params) {
  if (!params.selector && !params.text && !params.xpath) {
    throw new Error('selector, text, or xpath is required for click command');
  }

  let element = null;
  if (params.xpath) {
    element = evaluateXPath(params.xpath)[0];
  } else if (params.selector) {
    element = document.querySelector(params.selector);
  } else if (params.text) {
    element = findElementByText(params.text, params.tagName);
  }

  if (!element) {
    throw new Error('Element not found for click');
  }

  // Scroll into view
  element.scrollIntoView({ behavior: 'smooth', block: 'center' });
  await delay(300);

  // Simulate human-like click
  element.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
  await delay(100);
  element.click();
  element.dispatchEvent(new MouseEvent('mouseout', { bubbles: true }));

  await delay(500);
  return { success: true, tagName: element.tagName, text: element.textContent.substring(0, 100) };
}

/**
 * 3. Fill form field with human-like typing
 */
async function cmdFill(params) {
  if (!params.selector && !params.xpath) {
    throw new Error('selector or xpath is required for fill command');
  }
  if (params.value === undefined) {
    throw new Error('value is required for fill command');
  }

  let element = null;
  if (params.xpath) {
    element = evaluateXPath(params.xpath)[0];
  } else {
    element = document.querySelector(params.selector);
  }

  if (!element) {
    throw new Error('Element not found for fill');
  }

  // Focus on the element
  element.scrollIntoView({ behavior: 'smooth', block: 'center' });
  element.focus();
  await delay(200);

  // Clear existing value
  if (element.tagName === 'TEXTAREA' || (element.tagName === 'INPUT' && element.type !== 'checkbox')) {
    element.value = '';
  }

  // Type the value with human-like speed
  const value = String(params.value);
  const typeDelay = params.typeDelay || 50; // milliseconds per character

  for (const char of value) {
    element.value += char;
    element.dispatchEvent(new Event('input', { bubbles: true }));
    element.dispatchEvent(new Event('change', { bubbles: true }));
    await delay(typeDelay);
  }

  await delay(200);
  return { success: true, value: value.substring(0, 100), length: value.length };
}

/**
 * 4. Submit a form
 */
async function cmdSubmit(params) {
  if (!params.selector && !params.xpath) {
    throw new Error('selector or xpath is required for submit command');
  }

  let element = null;
  if (params.xpath) {
    element = evaluateXPath(params.xpath)[0];
  } else {
    element = document.querySelector(params.selector);
  }

  if (!element) {
    throw new Error('Element not found for submit');
  }

  // Find the form
  let form = element;
  if (element.tagName !== 'FORM') {
    form = element.closest('form');
  }

  if (!form) {
    throw new Error('No form found to submit');
  }

  form.submit();
  await delay(1000);
  return { success: true, formId: form.id, formName: form.name };
}

/**
 * 5. Extract text/attributes from element(s)
 */
async function cmdExtract(params) {
  if (!params.selector && !params.xpath) {
    throw new Error('selector or xpath is required for extract command');
  }

  let elements = [];
  if (params.xpath) {
    elements = evaluateXPath(params.xpath);
  } else {
    if (params.multiple) {
      elements = Array.from(document.querySelectorAll(params.selector));
    } else {
      const elem = document.querySelector(params.selector);
      elements = elem ? [elem] : [];
    }
  }

  if (elements.length === 0) {
    throw new Error('No elements found for extraction');
  }

  const results = elements.map(elem => {
    const data = {};

    if (params.attribute) {
      data.attribute = elem.getAttribute(params.attribute);
    } else {
      data.text = elem.textContent.trim();
      data.html = elem.innerHTML;
    }

    if (params.attributes) {
      data.attributes = {};
      params.attributes.forEach(attr => {
        data.attributes[attr] = elem.getAttribute(attr);
      });
    }

    return data;
  });

  return params.multiple ? results : results[0];
}

/**
 * 6. Take a screenshot
 */
async function cmdScreenshot(params) {
  // Use the canvas API to capture the screenshot
  const canvas = await html2canvas(document.body, {
    allowTaint: true,
    useCORS: true,
    backgroundColor: '#ffffff'
  });

  const dataUrl = canvas.toDataURL('image/png');
  return { success: true, dataUrl, width: canvas.width, height: canvas.height };
}

/**
 * 7. Execute JavaScript in page context
 */
async function cmdExecute(params) {
  if (!params.code) {
    throw new Error('code is required for execute command');
  }

  // Create and execute a function
  const func = new Function(params.code);
  const result = await func();
  return { success: true, result };
}

/**
 * 8. Wait for element to appear
 */
async function cmdWaitFor(params) {
  if (!params.selector && !params.xpath) {
    throw new Error('selector or xpath is required for wait_for command');
  }

  const timeout = params.timeout || 10000;
  const startTime = Date.now();

  return new Promise((resolve, reject) => {
    const checkElement = () => {
      let element = null;
      if (params.xpath) {
        const results = evaluateXPath(params.xpath);
        element = results[0];
      } else {
        element = document.querySelector(params.selector);
      }

      if (element && isElementVisible(element)) {
        resolve({ success: true, found: true });
      } else if (Date.now() - startTime > timeout) {
        reject(new Error('Timeout waiting for element'));
      } else {
        setTimeout(checkElement, 100);
      }
    };

    checkElement();
  });
}

/**
 * 9. Scroll page or to element
 */
async function cmdScroll(params) {
  if (params.selector || params.xpath) {
    let element = null;
    if (params.xpath) {
      element = evaluateXPath(params.xpath)[0];
    } else {
      element = document.querySelector(params.selector);
    }

    if (!element) {
      throw new Error('Element not found for scroll');
    }

    element.scrollIntoView({ behavior: 'smooth', block: params.block || 'center' });
  } else if (params.y !== undefined) {
    window.scrollTo({ top: params.y, left: params.x || 0, behavior: 'smooth' });
  } else {
    throw new Error('selector, xpath, or y position required for scroll command');
  }

  await delay(500);
  return { success: true, scrollY: window.scrollY, scrollX: window.scrollX };
}

/**
 * 10. Get cookies
 */
async function cmdGetCookies(_params) {
  const cookies = document.cookie.split(';').map(cookie => {
    const [name, value] = cookie.split('=');
    return { name: name.trim(), value: value.trim() };
  });
  return { cookies };
}

/**
 * 11. Set cookies
 */
async function cmdSetCookies(params) {
  if (!Array.isArray(params.cookies)) {
    throw new Error('cookies array is required for set_cookies command');
  }

  params.cookies.forEach(cookie => {
    const { name, value, path = '/', maxAge = 86400 } = cookie;
    document.cookie = `${name}=${value}; path=${path}; max-age=${maxAge}`;
  });

  return { success: true, count: params.cookies.length };
}

/**
 * 12. Get page source
 */
async function cmdGetPageSource(_params) {
  return {
    html: document.documentElement.outerHTML,
    url: window.location.href,
    title: document.title
  };
}

/**
 * Helper: Get current page state
 */
function getPageState() {
  return {
    url: window.location.href,
    title: document.title,
    scrollY: window.scrollY,
    scrollX: window.scrollX
  };
}

/**
 * Helper: Evaluate XPath expression
 */
function evaluateXPath(xpath) {
  const result = document.evaluate(
    xpath,
    document,
    null,
    XPathResult.ORDERED_NODE_SNAPSHOT_TYPE,
    null
  );
  const elements = [];
  for (let i = 0; i < result.snapshotLength; i++) {
    elements.push(result.snapshotItem(i));
  }
  return elements;
}

/**
 * Helper: Find element by text content
 */
function findElementByText(text, tagName = '*') {
  const elements = document.querySelectorAll(tagName);
  for (const elem of elements) {
    if (elem.textContent.includes(text)) {
      return elem;
    }
  }
  return null;
}

/**
 * Helper: Check if element is visible
 */
function isElementVisible(element) {
  const style = window.getComputedStyle(element);
  return style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0';
}

/**
 * Helper: Delay execution
 */
function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

console.log('[Content] Content script loaded and ready');
