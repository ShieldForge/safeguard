import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { BuildOptionsSection } from "./BuildOptionsSection";
import { BuildMetadataSection } from "./BuildMetadataSection";
import { CodeSigningSection } from "./CodeSigningSection";
import { makeFormData } from "../../test/helpers";

describe("BuildOptionsSection", () => {
    it("renders section title", () => {
        render(
            <BuildOptionsSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(screen.getByText("Build Options")).toBeInTheDocument();
    });

    it("renders output filename input", () => {
        render(
            <BuildOptionsSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(
            screen.getByPlaceholderText(/safeguard-custom/),
        ).toBeInTheDocument();
    });
});

describe("BuildMetadataSection", () => {
    it("renders section title", () => {
        render(
            <BuildMetadataSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(screen.getByText("Build Metadata")).toBeInTheDocument();
    });

    it("renders version input", () => {
        render(
            <BuildMetadataSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(screen.getByPlaceholderText("1.0.0")).toBeInTheDocument();
    });

    it("renders build tag input", () => {
        render(
            <BuildMetadataSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(
            screen.getByPlaceholderText(/customer-name/),
        ).toBeInTheDocument();
    });
});

describe("CodeSigningSection", () => {
    it("renders section title", () => {
        render(
            <CodeSigningSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(screen.getByText("Code Signing")).toBeInTheDocument();
    });

    it("has code signing disabled by default", () => {
        render(
            <CodeSigningSection
                formData={makeFormData({ code_signing_enabled: "false" })}
                onChange={vi.fn()}
            />,
        );
        expect(screen.getByDisplayValue("Disabled")).toBeInTheDocument();
    });
});
