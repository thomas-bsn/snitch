from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_eligibility_preview import VoiceEligibilityPreview
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    comparable_min_seconds: float | Unset = UNSET,
    contributive_min_seconds: float | Unset = UNSET,
    max_clipped_ratio: float | Unset = UNSET,
    durations: list[float] | Unset = UNSET,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["comparable_min_seconds"] = comparable_min_seconds

    params["contributive_min_seconds"] = contributive_min_seconds

    params["max_clipped_ratio"] = max_clipped_ratio

    json_durations: list[float] | Unset = UNSET
    if not isinstance(durations, Unset):
        json_durations = durations


    params["durations"] = json_durations


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/voice-eligibility/preview",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceEligibilityPreview:
    if response.status_code == 200:
        response_200 = VoiceEligibilityPreview.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceEligibilityPreview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    comparable_min_seconds: float | Unset = UNSET,
    contributive_min_seconds: float | Unset = UNSET,
    max_clipped_ratio: float | Unset = UNSET,
    durations: list[float] | Unset = UNSET,

) -> Response[Error | VoiceEligibilityPreview]:
    """  Administrateurs uniquement. Décide les empreintes de Locuteur local de production sous la règle
    examinée et sous chaque durée, avec le calcul même de la requalification, dans une transaction en
    lecture seule.

    Args:
        comparable_min_seconds (float | Unset):
        contributive_min_seconds (float | Unset):
        max_clipped_ratio (float | Unset):
        durations (list[float] | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceEligibilityPreview]
     """


    kwargs = _get_kwargs(
        comparable_min_seconds=comparable_min_seconds,
contributive_min_seconds=contributive_min_seconds,
max_clipped_ratio=max_clipped_ratio,
durations=durations,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    comparable_min_seconds: float | Unset = UNSET,
    contributive_min_seconds: float | Unset = UNSET,
    max_clipped_ratio: float | Unset = UNSET,
    durations: list[float] | Unset = UNSET,

) -> Error | VoiceEligibilityPreview | None:
    """  Administrateurs uniquement. Décide les empreintes de Locuteur local de production sous la règle
    examinée et sous chaque durée, avec le calcul même de la requalification, dans une transaction en
    lecture seule.

    Args:
        comparable_min_seconds (float | Unset):
        contributive_min_seconds (float | Unset):
        max_clipped_ratio (float | Unset):
        durations (list[float] | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceEligibilityPreview
     """


    return sync_detailed(
        client=client,
comparable_min_seconds=comparable_min_seconds,
contributive_min_seconds=contributive_min_seconds,
max_clipped_ratio=max_clipped_ratio,
durations=durations,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    comparable_min_seconds: float | Unset = UNSET,
    contributive_min_seconds: float | Unset = UNSET,
    max_clipped_ratio: float | Unset = UNSET,
    durations: list[float] | Unset = UNSET,

) -> Response[Error | VoiceEligibilityPreview]:
    """  Administrateurs uniquement. Décide les empreintes de Locuteur local de production sous la règle
    examinée et sous chaque durée, avec le calcul même de la requalification, dans une transaction en
    lecture seule.

    Args:
        comparable_min_seconds (float | Unset):
        contributive_min_seconds (float | Unset):
        max_clipped_ratio (float | Unset):
        durations (list[float] | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceEligibilityPreview]
     """


    kwargs = _get_kwargs(
        comparable_min_seconds=comparable_min_seconds,
contributive_min_seconds=contributive_min_seconds,
max_clipped_ratio=max_clipped_ratio,
durations=durations,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    comparable_min_seconds: float | Unset = UNSET,
    contributive_min_seconds: float | Unset = UNSET,
    max_clipped_ratio: float | Unset = UNSET,
    durations: list[float] | Unset = UNSET,

) -> Error | VoiceEligibilityPreview | None:
    """  Administrateurs uniquement. Décide les empreintes de Locuteur local de production sous la règle
    examinée et sous chaque durée, avec le calcul même de la requalification, dans une transaction en
    lecture seule.

    Args:
        comparable_min_seconds (float | Unset):
        contributive_min_seconds (float | Unset):
        max_clipped_ratio (float | Unset):
        durations (list[float] | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceEligibilityPreview
     """


    return (await asyncio_detailed(
        client=client,
comparable_min_seconds=comparable_min_seconds,
contributive_min_seconds=contributive_min_seconds,
max_clipped_ratio=max_clipped_ratio,
durations=durations,

    )).parsed
