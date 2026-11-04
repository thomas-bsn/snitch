import { Component, type ErrorInfo, type ReactNode } from "react";
import { ErrorAlert } from "./Alert";
import { useI18n } from "../i18n";

function Failure({ error }: { error: Error }) {
  const { t } = useI18n();
  return (
    <div className="flex flex-col gap-2">
      <ErrorAlert>{t("renderFailure.message")}</ErrorAlert>
      <p className="text-xs text-mute">
        <code>{error.message}</code>
      </p>
    </div>
  );
}

type Props = { children: ReactNode; resetKey?: string };
type State = { error?: Error };

/** Keeps a rendering exception inside its screen: a message and the cause instead of a blank page. */
export class RenderBoundary extends Component<Props, State> {
  state: State = {};

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error(error, info.componentStack);
  }

  componentDidUpdate(previous: Props) {
    if (this.state.error && previous.resetKey !== this.props.resetKey) {
      this.setState({ error: undefined });
    }
  }

  render() {
    return this.state.error ? (
      <Failure error={this.state.error} />
    ) : (
      this.props.children
    );
  }
}
