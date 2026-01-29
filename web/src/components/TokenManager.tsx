import { useEffect, useState } from "react";
import { getToken, setToken } from "../lib/api";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

export function TokenManager() {
  const [token, setTokenState] = useState("");
  const [stored, setStored] = useState(false);

  useEffect(() => {
    const existing = getToken();
    if (existing) {
      setTokenState(existing);
      setStored(true);
    }
  }, []);

  const save = () => {
    setToken(token.trim());
    setStored(Boolean(token.trim()));
  };

  const clear = () => {
    setTokenState("");
    setToken("");
    setStored(false);
  };

  return (
    <div className="flex w-full max-w-md items-center gap-2">
      <Input
        value={token}
        onChange={(event) => setTokenState(event.target.value)}
        placeholder="Bearer token for write actions"
        type="password"
      />
      <Button variant="secondary" size="sm" onClick={save}>
        {stored ? "Update" : "Save"}
      </Button>
      {stored && (
        <Button variant="ghost" size="sm" onClick={clear}>
          Clear
        </Button>
      )}
    </div>
  );
}
