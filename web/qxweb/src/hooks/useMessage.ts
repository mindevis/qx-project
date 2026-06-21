import { App } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';

/** Context-aware toast API (requires antd `<App>` in ThemeProvider). */
export function useMessage(): MessageInstance {
  return App.useApp().message;
}
