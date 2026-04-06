import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BuildPage } from "./BuildPage";

// Mock fetch globally
beforeEach(() => {
    vi.stubGlobal(
        "fetch",
        vi.fn().mockResolvedValue({
            ok: true,
            json: () =>
                Promise.resolve({
                    status: "ready",
                    info: ["Go: 1.22.0", "GCC: 13.2.0"],
                    source: "/src",
                    work: "/work",
                    output: "/output",
                }),
        }),
    );
});

describe("BuildPage", () => {
    it("renders the page header", () => {
        render(<BuildPage />);
        expect(screen.getByText("Configure Build")).toBeInTheDocument();
    });

    it("renders the sidebar brand", () => {
        render(<BuildPage />);
        expect(screen.getByText("Safeguard Mount")).toBeInTheDocument();
        expect(screen.getByText("Custom Build Server")).toBeInTheDocument();
    });

    it("renders the build button in the sidebar", () => {
        render(<BuildPage />);
        expect(screen.getByText("Build Binary")).toBeInTheDocument();
    });

    it("renders the main build button in the form", () => {
        render(<BuildPage />);
        expect(screen.getByText("Build Custom Binary")).toBeInTheDocument();
    });

    it("renders all major form sections", () => {
        render(<BuildPage />);
        expect(
            screen.getByText(/Connection.*Authentication/),
        ).toBeInTheDocument();
        // Use getAllByText for labels that appear in both sidebar steps and sections
        expect(screen.getAllByText("Target Platform").length).toBeGreaterThan(0);
        expect(screen.getAllByText("Code Signing").length).toBeGreaterThan(0);
    });

    it("renders step progress in sidebar", () => {
        render(<BuildPage />);
        // Step labels are unique to the sidebar StepProgress
        expect(screen.getByText("Connection & Auth")).toBeInTheDocument();
        expect(screen.getByText("Customize Options")).toBeInTheDocument();
        expect(screen.getByText("Build")).toBeInTheDocument();
    });

    it("can interact with form fields", async () => {
        const user = userEvent.setup();
        render(<BuildPage />);
        const vaultInput = screen.getByPlaceholderText("http://127.0.0.1:8200");
        await user.type(vaultInput, "http://vault:8200");
        expect(vaultInput).toHaveValue("http://vault:8200");
    });

    it("loads api/validate on mount", () => {
        render(<BuildPage />);
        expect(fetch).toHaveBeenCalledWith("/api/validate");
    });
});
