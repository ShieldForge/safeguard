import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MountPathsSection } from "./MountPathsSection";
import { makeFormData } from "../../test/helpers";

describe("MountPathsSection", () => {
    it("renders section title", () => {
        render(
            <MountPathsSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(screen.getByText(/Mount.*Paths/)).toBeInTheDocument();
    });

    it("renders mount point input", () => {
        render(
            <MountPathsSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(
            screen.getByPlaceholderText(/V:.*Windows/),
        ).toBeInTheDocument();
    });

    it("renders mapping config input", () => {
        render(
            <MountPathsSection formData={makeFormData()} onChange={vi.fn()} />,
        );
        expect(
            screen.getByPlaceholderText(/mapping\.json/),
        ).toBeInTheDocument();
    });

    it("shows current mount point value", () => {
        render(
            <MountPathsSection
                formData={makeFormData({ default_mount_point: "V:" })}
                onChange={vi.fn()}
            />,
        );
        expect(screen.getByDisplayValue("V:")).toBeInTheDocument();
    });
});
