#!/usr/bin/env python3
"""
Скрипт для автоматического исправления JSON тегов с snake_case на camelCase
в Go файлах из internal/models/

Использование:
    python scripts/fix_json_tags.py
"""

import re
import sys
import glob

# Полный список замен snake_case -> camelCase для JSON тегов
replacements = {
    "agent_id": "agentId",
    "allowed_ip_ranges": "allowedIpRanges",
    "assigned_to": "assignedTo",
    "auto_resolution_strategy": "autoResolutionStrategy",
    "auto_resolution_winner": "autoResolutionWinner",
    "avg_latency_ms": "avgLatencyMs",
    "carrier_id": "carrierId",
    "config_json": "configJson",
    "conflict_type": "conflictType",
    "country_id": "countryId",
    "created_at": "createdAt",
    "delivered_at": "deliveredAt",
    "denied_params": "deniedParams",
    "device_id": "deviceId",
    "error_message": "errorMessage",
    "entity_type": "entityType",
    "entity_id": "entityId",
    "execution_time_ms": "executionTimeMs",
    "failed_login_attempts": "failedLoginAttempts",
    "function_name": "functionName",
    "is_active": "isActive",
    "is_company": "isCompany",
    "is_enabled": "isEnabled",
    "is_refund_requested": "isRefundRequested",
    "is_verified": "isVerified",
    "item_id": "itemId",
    "last_activity_at": "lastActivityAt",
    "last_failure_at": "lastFailureAt",
    "last_full_sync_at": "lastFullSyncAt",
    "last_login": "lastLogin",
    "last_request_at": "lastRequestAt",
    "last_seen_at": "lastSeenAt",
    "last_success_at": "lastSuccessAt",
    "last_sync_at": "lastSyncAt",
    "last_sync_status": "lastSyncStatus",
    "last_synced_at": "lastSyncedAt",
    "last_updated": "lastUpdated",
    "location_dest_id": "locationDestId",
    "location_id": "locationId",
    "lot_id": "lotId",
    "max_rate_per_min": "maxRatePerMin",
    "max_tokens_per_day": "maxTokensPerDay",
    "mapped_location_id": "mappedLocationId",
    "picking_delivery_id": "pickingDeliveryId",
    "picking_id": "pickingId",
    "picking_type_id": "pickingTypeId",
    "package_type_id": "packageTypeId",
    "package_id": "packageId",
    "partner_id": "partnerId",
    "preferred_language": "preferredLanguage",
    "product_id": "productId",
    "records_conflicts": "recordsConflicts",
    "records_synced": "recordsSynced",
    "remote_data": "remoteData",
    "remote_metadata": "remoteMetadata",
    "request_data": "requestData",
    "require_approval": "requireApproval",
    "resolved_by": "resolvedBy",
    "resolved_at": "resolvedAt",
    "response_data": "responseData",
    "result_package_id": "resultPackageId",
    "retry_count": "retryCount",
    "rma_number": "rmaNumber",
    "rma_reason": "rmaReason",
    "rma_request_id": "rmaRequestId",
    "scheduled_at": "scheduledAt",
    "scheduled_date": "scheduledDate",
    "shipped_at": "shippedAt",
    "sort_order": "sortOrder",
    "source_instance": "sourceInstance",
    "started_at": "startedAt",
    "completed_at": "completedAt",
    "status_code": "statusCode",
    "sync_duration_ms": "syncDurationMs",
    "target_instance": "targetInstance",
    "technician_id": "technicianId",
    "total_tokens_used": "totalTokensUsed",
    "tracking_number": "trackingNumber",
    "updated_at": "updatedAt",
    "warehouse_id": "warehouseId",
    "window_start": "windowStart",
}


def fix_json_tags(content):
    """Заменяет snake_case на camelCase в json:"..." тегах"""

    # Ищем все json:"..." теги
    pattern = r'json:"([^"]*)"'

    def replace_tag(match):
        old_value = match.group(1)

        # Проверяем есть ли опции (omitempty)
        options = ""
        base_value = old_value
        if ",omitempty" in old_value:
            base_value = old_value.replace(",omitempty", "")
            options = ",omitempty"

        # Делаем замену если значение есть в списке
        if base_value in replacements:
            new_value = replacements[base_value]
            return f'json:"{new_value}{options}"'

        # Иначе оставляем как есть
        return match.group(0)

    return re.sub(pattern, replace_tag, content)


def main():
    """Основная функция скрипта"""
    print("Fixing JSON tags in models...\n")

    # Обрабатываем все go файлы в internal/models/
    go_files = glob.glob("internal/models/*.go")

    if not go_files:
        print("ERROR: No files found in internal/models/")
        sys.exit(1)

    updated_count = 0
    skipped_count = 0

    for filepath in sorted(go_files):
        try:
            with open(filepath, "r", encoding="utf-8") as f:
                content = f.read()

            new_content = fix_json_tags(content)

            if new_content != content:
                with open(filepath, "w", encoding="utf-8") as f:
                    f.write(new_content)
                print(f"[OK] Updated: {filepath}")
                updated_count += 1
            else:
                print(f"[SKIP] {filepath}")
                skipped_count += 1

        except Exception as e:
            print(f"[ERROR] Failed to process {filepath}: {e}")
            sys.exit(1)

    print(f"\nSummary:")
    print(f"   Updated files: {updated_count}")
    print(f"   Skipped files: {skipped_count}")
    print(f"\n[OK] JSON tags are now in camelCase!")


if __name__ == "__main__":
    main()
