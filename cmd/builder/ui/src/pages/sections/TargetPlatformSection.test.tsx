import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { TargetPlatformSection } from "./TargetPlatformSection";
import { makeFormData } from "../../test/helpers";

describe("TargetPlatformSection", () => {
    it("renders target platform section title", () => {
        render(
            <TargetPlatformSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(screen.getByText("Target Platform")).toBeInTheDocument();
    });

    it("renders OS select with options", () => {
        render(
            <TargetPlatformSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(screen.getByText("Windows")).toBeInTheDocument();
        expect(screen.getByText("Linux")).toBeInTheDocument();
        expect(screen.getByText("macOS")).toBeInTheDocument();
    });

    it("renders architecture select with options", () => {
        render(
            <TargetPlatformSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(screen.getByText("amd64 (64-bit)")).toBeInTheDocument();
        expect(
            screen.getByText("arm64 (Apple Silicon, ARM64)"),
        ).toBeInTheDocument();
        expect(screen.getByText("386 (32-bit)")).toBeInTheDocument();
    });

    it("reflects current form values", () => {
        render(
            <TargetPlatformSection
                formData={makeFormData({ target_os: "linux", target_arch: "arm64" })}
                onChange={vi.fn()}
            />,
        );
        const osSelect = screen.getByDisplayValue("Linux");
        expect(osSelect).toBeInTheDocument();
    });
});
