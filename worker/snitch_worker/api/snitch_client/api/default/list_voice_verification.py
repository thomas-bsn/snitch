from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.list_voice_verification_level import ListVoiceVerificationLevel
from ...models.voice_verification import VoiceVerification
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    *,
    level: ListVoiceVerificationLevel | Unset = ListVoiceVerificationLevel.ALL,
    clip: int | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 20,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    json_level: str | Unset = UNSET
    if not isinstance(level, Unset):
        json_level = level.value

    params["level"] = json_level

    params["clip"] = clip

    params["page"] = page

    params["page_size"] = page_size


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/voice-verification",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceVerification:
    if response.status_code == 200:
        response_200 = VoiceVerification.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceVerification]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    level: ListVoiceVerificationLevel | Unset = ListVoiceVerificationLevel.ALL,
    clip: int | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 20,

) -> Response[Error | VoiceVerification]:
    """  Utilisateurs connectés. Vue de vérification : uniquement les voix inconnues ou ambiguës (avec leurs
    Personnes candidates), sur les clips visibles. Tri par fréquence : clips distincts du groupe proposé
    de la voix, puis nombre de voix du groupe, puis temps de parole. Trancher un cas est une validation
    humaine (POST /api/local-speakers/{id}/validation) : cette lecture n'écrit rien. Les compteurs par
    niveau sont calculés en base. Aucune promesse de qualité n'est dérivée du nombre d'extraits.

    Args:
        level (ListVoiceVerificationLevel | Unset):  Default: ListVoiceVerificationLevel.ALL.
        clip (int | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceVerification]
     """


    kwargs = _get_kwargs(
        level=level,
clip=clip,
page=page,
page_size=page_size,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    level: ListVoiceVerificationLevel | Unset = ListVoiceVerificationLevel.ALL,
    clip: int | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 20,

) -> Error | VoiceVerification | None:
    """  Utilisateurs connectés. Vue de vérification : uniquement les voix inconnues ou ambiguës (avec leurs
    Personnes candidates), sur les clips visibles. Tri par fréquence : clips distincts du groupe proposé
    de la voix, puis nombre de voix du groupe, puis temps de parole. Trancher un cas est une validation
    humaine (POST /api/local-speakers/{id}/validation) : cette lecture n'écrit rien. Les compteurs par
    niveau sont calculés en base. Aucune promesse de qualité n'est dérivée du nombre d'extraits.

    Args:
        level (ListVoiceVerificationLevel | Unset):  Default: ListVoiceVerificationLevel.ALL.
        clip (int | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceVerification
     """


    return sync_detailed(
        client=client,
level=level,
clip=clip,
page=page,
page_size=page_size,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    level: ListVoiceVerificationLevel | Unset = ListVoiceVerificationLevel.ALL,
    clip: int | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 20,

) -> Response[Error | VoiceVerification]:
    """  Utilisateurs connectés. Vue de vérification : uniquement les voix inconnues ou ambiguës (avec leurs
    Personnes candidates), sur les clips visibles. Tri par fréquence : clips distincts du groupe proposé
    de la voix, puis nombre de voix du groupe, puis temps de parole. Trancher un cas est une validation
    humaine (POST /api/local-speakers/{id}/validation) : cette lecture n'écrit rien. Les compteurs par
    niveau sont calculés en base. Aucune promesse de qualité n'est dérivée du nombre d'extraits.

    Args:
        level (ListVoiceVerificationLevel | Unset):  Default: ListVoiceVerificationLevel.ALL.
        clip (int | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceVerification]
     """


    kwargs = _get_kwargs(
        level=level,
clip=clip,
page=page,
page_size=page_size,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    level: ListVoiceVerificationLevel | Unset = ListVoiceVerificationLevel.ALL,
    clip: int | Unset = UNSET,
    page: int | Unset = 1,
    page_size: int | Unset = 20,

) -> Error | VoiceVerification | None:
    """  Utilisateurs connectés. Vue de vérification : uniquement les voix inconnues ou ambiguës (avec leurs
    Personnes candidates), sur les clips visibles. Tri par fréquence : clips distincts du groupe proposé
    de la voix, puis nombre de voix du groupe, puis temps de parole. Trancher un cas est une validation
    humaine (POST /api/local-speakers/{id}/validation) : cette lecture n'écrit rien. Les compteurs par
    niveau sont calculés en base. Aucune promesse de qualité n'est dérivée du nombre d'extraits.

    Args:
        level (ListVoiceVerificationLevel | Unset):  Default: ListVoiceVerificationLevel.ALL.
        clip (int | Unset):
        page (int | Unset):  Default: 1.
        page_size (int | Unset):  Default: 20.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceVerification
     """


    return (await asyncio_detailed(
        client=client,
level=level,
clip=clip,
page=page,
page_size=page_size,

    )).parsed
