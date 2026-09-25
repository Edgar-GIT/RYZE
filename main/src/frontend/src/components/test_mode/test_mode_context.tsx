import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode
} from "react";

import {
  enterTestMode as enterTestModeApi,
  exitTestMode as exitTestModeApi,
  fetchTestModeStatus,
  type TestModePersona,
  type TestModeStatus
} from "@/services/test_mode_api";

/**
 * TestMode carries the active Admin Test Mode session to every client page.
 * The state is always refreshed from the public status endpoint so a stale
 * session (expired token, exited elsewhere) is reflected promptly, and is
 * only ever treated as active when the server reports it.
 *
 * Entering and exiting are full page navigations: the backend swaps the
 * cookies (admin ↔ persona access token), so a hard redirect guarantees the
 * client picks up the exact session the server minted.
 */
interface TestModeValue {
  status: TestModeStatus | null;
  isLoading: boolean;
  isActive: boolean;
  persona: TestModePersona | null;
  errorMessage: string;
  enterTestMode: (persona: TestModePersona, returnPath: string) => Promise<void>;
  exitTestMode: () => Promise<void>;
}

const TestModeContext = createContext<TestModeValue | null>(null);

export const useTestMode = (): TestModeValue => {
  const value = useContext(TestModeContext);
  if (!value) {
    throw new Error("useTestMode must be used within TestModeProvider");
  }
  return value;
};

export const TestModeProvider = ({ children }: { children: ReactNode }) => {
  const [status, setStatus] = useState<TestModeStatus | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const refresh = useCallback(async () => {
    setIsLoading(true);
    setErrorMessage("");
    try {
      setStatus(await fetchTestModeStatus());
    } catch {
      setStatus({ active: false });
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const enterTestMode = useCallback(async (persona: TestModePersona, returnPath: string) => {
    setErrorMessage("");
    try {
      const result = await enterTestModeApi(persona, returnPath);
      const destination = result.return_path || "/";
      window.location.assign(destination);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unable to enter Test Mode.";
      setStatus({ active: false });
      setErrorMessage(message);
      throw new Error(message);
    }
  }, []);

  const exitTestMode = useCallback(async () => {
    setErrorMessage("");
    try {
      const result = await exitTestModeApi();
      window.location.assign(result.return_path || "/");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unable to exit Test Mode.";
      setErrorMessage(message);
      throw new Error(message);
    }
  }, []);

  const isActive = status?.active === true;
  const persona =
    isActive && status.persona === "trainer"
      ? "trainer"
      : isActive && status.persona === "client"
        ? "client"
        : null;

  return (
    <TestModeContext.Provider
      value={{
        status,
        isLoading,
        isActive,
        persona,
        errorMessage,
        enterTestMode,
        exitTestMode
      }}
    >
      {children}
    </TestModeContext.Provider>
  );
};