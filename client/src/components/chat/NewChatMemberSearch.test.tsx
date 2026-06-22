import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { NewChatMemberSearch } from "@/components/chat/NewChatMemberSearch";

const searchUsers = vi.fn();

vi.mock("@/lib/users", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/users")>();
  return {
    ...actual,
    searchUsers: (...args: unknown[]) => searchUsers(...args),
  };
});

describe("NewChatMemberSearch", () => {
  it("disables search below minimum query length", () => {
    render(
      <NewChatMemberSearch
        currentUserId="user-1"
        members={[]}
        onAddMember={vi.fn()}
        onBack={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText(/Search/i), { target: { value: "ab" } });
    expect(screen.getByRole("button", { name: "Search" })).toBeDisabled();
  });

  it("filters out current user and existing members", async () => {
    searchUsers.mockResolvedValue([
      {
        id: "user-1",
        name: "You",
        oidcProviderId: "p1",
        subject: "s1",
        createdAt: "2026-06-11T12:00:00.000Z",
        updatedAt: "2026-06-11T12:00:00.000Z",
      },
      {
        id: "member-1",
        name: "Bob",
        oidcProviderId: "p1",
        subject: "s2",
        createdAt: "2026-06-11T12:00:00.000Z",
        updatedAt: "2026-06-11T12:00:00.000Z",
      },
      {
        id: "member-2",
        name: "Carol",
        oidcProviderId: "p1",
        subject: "s3",
        createdAt: "2026-06-11T12:00:00.000Z",
        updatedAt: "2026-06-11T12:00:00.000Z",
      },
    ]);

    render(
      <NewChatMemberSearch
        currentUserId="user-1"
        members={[{ id: "member-1", name: "Bob", oidcProviderId: "p1", subject: "s2", createdAt: "", updatedAt: "" }]}
        onAddMember={vi.fn()}
        onBack={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText(/Search/i), { target: { value: "bob" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));

    await waitFor(() => {
      expect(searchUsers).toHaveBeenCalledWith("bob");
    });
    expect(screen.getByText("Carol")).toBeInTheDocument();
    expect(screen.queryByText("You")).not.toBeInTheDocument();
    expect(screen.queryByText("Bob")).not.toBeInTheDocument();
  });

  it("shows empty state and error state", async () => {
    searchUsers.mockResolvedValueOnce([]);
    render(
      <NewChatMemberSearch
        currentUserId="user-1"
        members={[]}
        onAddMember={vi.fn()}
        onBack={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText(/Search/i), { target: { value: "zzz" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    expect(await screen.findByText("No users found.")).toBeInTheDocument();

    searchUsers.mockRejectedValueOnce(new Error("failed"));
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not search users. Try again.");
  });

  it("shows a truncated user id when the result has no name", async () => {
    searchUsers.mockResolvedValue([
      {
        id: "99999999-9999-9999-9999-999999999999/11111111-1111-1111-1111-111111111111",
        oidcProviderId: "p1",
        subject: "s4",
        createdAt: "2026-06-11T12:00:00.000Z",
        updatedAt: "2026-06-11T12:00:00.000Z",
      },
    ]);

    render(
      <NewChatMemberSearch
        currentUserId="user-1"
        members={[]}
        onAddMember={vi.fn()}
        onBack={vi.fn()}
      />,
    );

    fireEvent.change(screen.getByLabelText(/Search/i), { target: { value: "1111" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));

    expect(await screen.findByText("99999999...111111111111")).toBeInTheDocument();
    expect(screen.queryByText("Add name")).not.toBeInTheDocument();
  });

  it("calls onAddMember and onBack", async () => {
    searchUsers.mockResolvedValue([
      {
        id: "member-2",
        name: "Carol",
        oidcProviderId: "p1",
        subject: "s3",
        createdAt: "2026-06-11T12:00:00.000Z",
        updatedAt: "2026-06-11T12:00:00.000Z",
      },
    ]);
    const onAddMember = vi.fn();
    const onBack = vi.fn();

    render(
      <NewChatMemberSearch
        currentUserId="user-1"
        members={[]}
        onAddMember={onAddMember}
        onBack={onBack}
      />,
    );

    fireEvent.change(screen.getByLabelText(/Search/i), { target: { value: "car" } });
    fireEvent.click(screen.getByRole("button", { name: "Search" }));
    fireEvent.click(await screen.findByRole("button", { name: "Add" }));
    expect(onAddMember).toHaveBeenCalledWith(expect.objectContaining({ id: "member-2" }));

    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(onBack).toHaveBeenCalled();
  });
});
