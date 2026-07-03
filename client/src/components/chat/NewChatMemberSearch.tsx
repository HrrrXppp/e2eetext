import { FormEvent, useState } from "react";
import { canSearchUsers, searchUsers, userLabel, type User } from "@/lib/users";

type NewChatMemberSearchProps = {
  currentUserId: string;
  members: User[];
  onAddMember: (member: User) => void;
  onBack: () => void;
};

export function NewChatMemberSearch({
  currentUserId,
  members,
  onAddMember,
  onBack,
}: NewChatMemberSearchProps) {
  const [query, setQuery] = useState("");
  const [searchResults, setSearchResults] = useState<User[]>([]);
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [hasSearched, setHasSearched] = useState(false);

  async function handleSearch(event?: FormEvent) {
    event?.preventDefault();

    const trimmedQuery = query.trim();
    if (!canSearchUsers(trimmedQuery)) {
      setSearchResults([]);
      setSearchError(null);
      setHasSearched(false);
      return;
    }

    setSearching(true);
    setSearchError(null);

    const excluded = new Set([currentUserId, ...members.map((member) => member.id)]);

    try {
      const results = await searchUsers(trimmedQuery);
      setSearchResults(results.filter((item) => !excluded.has(item.id)));
      setHasSearched(true);
    } catch {
      setSearchResults([]);
      setSearchError("Could not search users. Try again.");
      setHasSearched(true);
    } finally {
      setSearching(false);
    }
  }

  return (
    <div className="new-chat-dialog__search">
      <button type="button" className="new-chat-dialog__back" onClick={onBack}>
        Back
      </button>

      <p className="new-chat-dialog__hint">
        Enter at least 3 characters. From 5 characters we also match user IDs.
      </p>

      <form className="new-chat-dialog__search-form" onSubmit={handleSearch}>
        <label className="new-chat-dialog__field">
          <span>Search</span>
          <input
            type="search"
            value={query}
            onChange={(event) => {
              setQuery(event.target.value);
              setHasSearched(false);
            }}
            placeholder="Name or user ID"
            autoComplete="off"
            autoFocus
          />
        </label>

        <button
          type="submit"
          className="new-chat-dialog__submit"
          disabled={searching || !canSearchUsers(query)}
        >
          {searching ? "Searching..." : "Search"}
        </button>
      </form>

      {searchError ? (
        <p className="new-chat-dialog__error" role="alert">
          {searchError}
        </p>
      ) : null}

      {hasSearched && !searching && !searchError && searchResults.length === 0 ? (
        <p className="new-chat-dialog__status">No users found.</p>
      ) : null}

      {searchResults.length > 0 ? (
        <ul className="new-chat-dialog__results" aria-label="Search results">
          {searchResults.map((member) => (
            <li key={member.id} className="new-chat-dialog__result">
              <div className="new-chat-dialog__result-copy">
                <span className="new-chat-dialog__result-name">
                  {userLabel(member.name, member.id)}
                </span>
                <span className="new-chat-dialog__result-id">{member.id}</span>
              </div>
              <button
                type="button"
                className="new-chat-dialog__add"
                onClick={() => onAddMember(member)}
              >
                Add
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
